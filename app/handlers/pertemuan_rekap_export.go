package handlers

import (
 "bytes"
 "encoding/csv"
 "fmt"
 "strconv"
 "strings"
 "time"

 "github.com/gofiber/fiber/v2"
 "github.com/maulanashalihin/laju-go/app/models"
)

// rekapDateRange is intentionally shared by export validation and tests.
// All dates are school calendar dates (YYYY-MM-DD), not JavaScript time zones.
type rekapDateRange struct { From, To, Label string }

func parseRekapDateRange(mode, month, day, from, to string, meetings []models.PertemuanResponse) (rekapDateRange, error) {
 date := func(s string) bool {
  t,err := time.Parse(time.DateOnly,s)
  return err == nil && t.Format(time.DateOnly) == s
 }
 switch mode {
 case "", "month":
  if month == "" {
   for _,p := range meetings { if date(p.Tanggal) && (month == "" || p.Tanggal[:7] > month) {month=p.Tanggal[:7]} }
   if month == "" {month=time.Now().In(time.FixedZone("WIB",7*3600)).Format("2006-01")}
  }
  t,err := time.Parse("2006-01",month)
  if err != nil || t.Format("2006-01") != month {return rekapDateRange{},fmt.Errorf("bulan tidak valid")}
  return rekapDateRange{From:month+"-01",To:t.AddDate(0,1,0).AddDate(0,0,-1).Format(time.DateOnly),Label:month},nil
 case "day":
  if !date(day) {return rekapDateRange{},fmt.Errorf("tanggal tidak valid")}
  return rekapDateRange{From:day,To:day,Label:day},nil
 case "range":
  if !date(from)||!date(to)||from>to {return rekapDateRange{},fmt.Errorf("rentang tanggal tidak valid")}
  a,_:=time.Parse(time.DateOnly,from)
  b,_:=time.Parse(time.DateOnly,to)
  if b.Sub(a).Hours()/24 > 365 {return rekapDateRange{},fmt.Errorf("rentang maksimal 366 hari")}
  return rekapDateRange{From:from,To:to,Label:from+"_"+to},nil
 default:
  return rekapDateRange{},fmt.Errorf("mode filter tidak valid")
 }
}

func filterRekapMeetings(rows []models.PertemuanResponse, selected rekapDateRange) []models.PertemuanResponse {
 out:=make([]models.PertemuanResponse,0)
 for _,p:=range rows {if p.Tanggal>=selected.From && p.Tanggal<=selected.To {out=append(out,p)}}
 return out
}

// Excel-style CSV is intentionally a detailed register (one row per student
// per meeting) so a note/status can be audited, not only a summary aggregate.
// Notes remain CSV quoted, and spreadsheet formulas are neutralized.
func csvSafeRekap(s string) string {
 s = strings.TrimSpace(s)
 if s == "" {return ""}
 trimmed:=strings.TrimLeft(s," \t\r\n")
 if len(trimmed)>0 && strings.ContainsRune("=+-@",rune(trimmed[0])) {return "'"+s}
 return s
}

func writeRekapCSV(buf *bytes.Buffer, className string, santri []models.SantriGuruResponse, meetings []models.PertemuanResponse, attendance map[string]models.AbsensiResponse) error {
 buf.WriteString("\xef\xbb\xbf") // UTF-8 BOM for Excel
 w:=csv.NewWriter(buf)
 if err:=w.Write([]string{"Kelas","Bulan","Tanggal","Pertemuan","Level","ID Santri","Nama Santri","Status","Catatan","Batas Materi"});err!=nil{return err}
 for _,p:=range meetings {
  for _,s:=range santri {
   a,ok:=attendance[fmt.Sprintf("%d:%d",s.ID,p.ID)]
   status:="Belum dicatat"
   if ok {status=a.Status}
   if err:=w.Write([]string{csvSafeRekap(className),p.Tanggal[:7],p.Tanggal,strconv.FormatInt(p.PertemuanKe,10),csvSafeRekap(p.LevelNama),csvSafeRekap(s.IDMahasantri),csvSafeRekap(s.Nama),status,csvSafeRekap(a.Catatan),csvSafeRekap(a.BatasMateri)});err!=nil{return err}
  }
 }
 w.Flush()
 return w.Error()
}

// ExportRekapAbsensi uses exactly the same class authorization as the page.
// It is read-only and never mutates attendance records.
func (h *PertemuanHandler) ExportRekapAbsensi(c *fiber.Ctx) error {
 kelasID,err:=strconv.ParseInt(c.Params("id"),10,64)
 if err!=nil||kelasID<=0 {return c.SendStatus(fiber.StatusBadRequest)}
 if _,err:=h.ensureClassAccess(c,kelasID);err!=nil {return c.SendStatus(fiber.StatusForbidden)}
 kelas,_,err:=h.guruService.GetDetailKelasForViewer(nil,kelasID)
 if err!=nil {return c.SendStatus(fiber.StatusNotFound)}
 santri,all,absensi,err:=h.pertemuanService.GetRekapAbsensi(kelasID)
 if err!=nil {return c.SendStatus(fiber.StatusInternalServerError)}
 sel,err:=parseRekapDateRange(c.Query("mode"),c.Query("month"),c.Query("date"),c.Query("from"),c.Query("to"),all)
 if err!=nil {return c.Status(fiber.StatusBadRequest).SendString(err.Error())}
 meetings:=filterRekapMeetings(all,sel)
 var buf bytes.Buffer
 if err:=writeRekapCSV(&buf,kelas.NamaKelas,santri,meetings,absensi);err!=nil {return c.SendStatus(fiber.StatusInternalServerError)}
 c.Set(fiber.HeaderContentType,"text/csv; charset=utf-8")
 c.Set(fiber.HeaderContentDisposition,fmt.Sprintf("attachment; filename=\"rekap-absensi-kelas-%d-%s.csv\"",kelasID,sel.Label))
 c.Set(fiber.HeaderCacheControl,"no-store")
 return c.Send(buf.Bytes())
}
