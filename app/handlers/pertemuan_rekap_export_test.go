package handlers

import (
 "bytes"
 "encoding/csv"
 "strings"
 "testing"

 "github.com/maulanashalihin/laju-go/app/models"
 "github.com/stretchr/testify/require"
)

func TestRekapPeriodFilters(t *testing.T) {
 meetings:=[]models.PertemuanResponse{
  {ID:11,Tanggal:"2026-09-29",PertemuanKe:1},
  {ID:12,Tanggal:"2026-10-03",PertemuanKe:2},
  {ID:13,Tanggal:"2026-10-24",PertemuanKe:3},
  {ID:14,Tanggal:"2026-11-01",PertemuanKe:4},
 }
 cases:=[]struct{
  name,mode,month,day,from,to string
  want []int64
  bad bool
 }{
  {name:"latest month default",mode:"month",want:[]int64{14}},
  {name:"october",mode:"month",month:"2026-10",want:[]int64{12,13}},
  {name:"specific date",mode:"day",day:"2026-10-24",want:[]int64{13}},
  {name:"custom crosses months",mode:"range",from:"2026-09-29",to:"2026-10-03",want:[]int64{11,12}},
  {name:"valid empty month",mode:"month",month:"2026-08",want:[]int64{}},
  {name:"invalid month",mode:"month",month:"2026-13",bad:true},
  {name:"invalid day",mode:"day",day:"2026-02-30",bad:true},
  {name:"reversed",mode:"range",from:"2026-10-24",to:"2026-10-03",bad:true},
  {name:"invalid mode",mode:"all",bad:true},
 }
 for _,tc:=range cases {t.Run(tc.name,func(t *testing.T){
  selected,err:=parseRekapDateRange(tc.mode,tc.month,tc.day,tc.from,tc.to,meetings)
  if tc.bad {require.Error(t,err);return}
  require.NoError(t,err)
  got:=[]int64{}
  for _,p:=range filterRekapMeetings(meetings,selected){got=append(got,p.ID)}
  require.Equal(t,tc.want,got)
 })}
}

func TestRekapCSVIncludesPeriodNotesAndNeutralizesFormulas(t *testing.T) {
 santri:=[]models.SantriGuruResponse{{ID:21,Nama:"=cmd()",IDMahasantri:"RS-21"},{ID:22,Nama:"Ali",IDMahasantri:"RS-22"}}
 meetings:=[]models.PertemuanResponse{{ID:5,Tanggal:"2026-10-03",PertemuanKe:2,LevelNama:"Tahsin"},{ID:6,Tanggal:"2026-11-01",PertemuanKe:3,LevelNama:"Tahsin"}}
 attendance:=map[string]models.AbsensiResponse{
  "21:5":{Status:"hadir",Catatan:"Ulang, huruf ح",BatasMateri:"+Al Fatihah"},
  "22:5":{Status:"telat",Catatan:"Datang terlambat"},
 }
 sel,err:=parseRekapDateRange("month","2026-10","","","",meetings)
 require.NoError(t,err)
 var buf bytes.Buffer
 require.NoError(t,writeRekapCSV(&buf,"Kelas A",santri,filterRekapMeetings(meetings,sel),attendance))
 require.True(t,strings.HasPrefix(buf.String(),"\xef\xbb\xbf"))
 r:=csv.NewReader(strings.NewReader(strings.TrimPrefix(buf.String(),"\xef\xbb\xbf")))
 rows,err:=r.ReadAll()
 require.NoError(t,err)
 require.Len(t,rows,3) // header + 2 students, October only
 require.Equal(t,"Bulan",rows[0][1])
 require.Equal(t,"2026-10",rows[1][1])
 require.Equal(t,"2026-10-03",rows[1][2])
 require.Equal(t,"'=cmd()",rows[1][6])
 require.Equal(t,"hadir",rows[1][7])
 require.Equal(t,"Ulang, huruf ح",rows[1][8])
 require.Equal(t,"'+Al Fatihah",rows[1][9])
 require.Equal(t,"telat",rows[2][7])
}
