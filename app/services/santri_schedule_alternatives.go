package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/maulanashalihin/laju-go/app/queries"
)

var (
	ErrAlternativeInvalid = errors.New("alternatif jadwal tidak valid")
	ErrAlternativeUnavailable = errors.New("alternatif jadwal tidak tersedia atau santri bukan onboarding aktif")
)

type SantriScheduleAlternativeInput struct {
	JadwalKode string `json:"jadwal_kode"`
	Status string `json:"status"`
	ConsentChannel string `json:"consent_channel"`
	ConsentNote string `json:"consent_note"`
}

type SantriScheduleAlternativeService struct {
	q *queries.Querier
}

func NewSantriScheduleAlternativeService(q *queries.Querier) *SantriScheduleAlternativeService {
	return &SantriScheduleAlternativeService{q:q}
}

func validateAlternative(req *SantriScheduleAlternativeInput) error {
	req.JadwalKode=strings.TrimSpace(req.JadwalKode)
	req.ConsentNote=strings.TrimSpace(req.ConsentNote)
	req.ConsentChannel=strings.TrimSpace(strings.ToLower(req.ConsentChannel))
	if req.JadwalKode==""||len(req.JadwalKode)>100||len(req.ConsentNote)>500{
		return fmt.Errorf("%w: kode dan catatan wajib valid",ErrAlternativeInvalid)
	}
	switch req.Status{
	case "proposed":
		if req.ConsentChannel!="" {return fmt.Errorf("%w: usulan belum dapat memiliki channel konfirmasi",ErrAlternativeInvalid)}
	case "approved":
		if req.ConsentChannel!="whatsapp"&&req.ConsentChannel!="phone"&&req.ConsentChannel!="other"{
			return fmt.Errorf("%w: sumber persetujuan harus diisi",ErrAlternativeInvalid)
		}
		if req.ConsentNote==""{return fmt.Errorf("%w: catat bukti atau rujukan persetujuan santri",ErrAlternativeInvalid)}
	case "revoked":
		if req.ConsentNote==""{return fmt.Errorf("%w: alasan pencabutan wajib diisi",ErrAlternativeInvalid)}
		if req.ConsentChannel!="" {return fmt.Errorf("%w: sumber persetujuan tidak boleh diisi saat mencabut",ErrAlternativeInvalid)}
	default:return fmt.Errorf("%w: status tidak dikenal",ErrAlternativeInvalid)
	}
	return nil
}

func (s *SantriScheduleAlternativeService) List(ctx context.Context,santriID int64) ([]queries.ScheduleAlternativeOffer,[]queries.SantriScheduleAlternative,error) {
	if santriID<=0{return nil,nil,ErrAlternativeInvalid}
	offers,err:=s.q.ListAlternateOffersForSantri(ctx,santriID)
	if err!=nil{return nil,nil,err}
	rows,err:=s.q.ListSantriScheduleAlternatives(ctx,santriID)
	return offers,rows,err
}

// Save only stores an attested choice. It never assigns a student, changes
// the primary schedule, reserves teacher time or publishes an offer.
func (s *SantriScheduleAlternativeService) Save(ctx context.Context,id,actor int64,input SantriScheduleAlternativeInput) error {
	if id<=0||actor<=0{return ErrAlternativeInvalid}
	if err:=validateAlternative(&input);err!=nil{return err}
	tx,err:=s.q.BeginTx(ctx)
	if err!=nil{return err}
	defer tx.Rollback()
	q:=s.q.WithTx(tx)
	oldID,oldStatus,oldChannel,oldNote,err:=q.GetSantriAlternativeState(ctx,id,input.JadwalKode)
	if err!=nil&&!errors.Is(err,sql.ErrNoRows){return err}
	if errors.Is(err,sql.ErrNoRows){oldID=0;oldStatus="";oldChannel="";oldNote=""}
	if input.Status=="revoked"{
		if oldID==0{return ErrAlternativeUnavailable}
	}else{
		allowed,err:=q.CanRecordSantriAlternative(ctx,id,input.JadwalKode)
		if err!=nil{return err}
		if !allowed{return ErrAlternativeUnavailable}
	}
	if oldStatus==input.Status&&oldChannel==input.ConsentChannel&&oldNote==input.ConsentNote{return nil}
	newID,err:=q.SaveSantriAlternativeTx(ctx,id,input.JadwalKode,input.Status,input.ConsentChannel,input.ConsentNote,actor)
	if err!=nil{return err}
	if err:=q.AppendSantriAlternativeEventTx(ctx,newID,id,actor,oldStatus,input.Status,input.ConsentChannel,input.ConsentNote);err!=nil{return err}
	return tx.Commit()
}
