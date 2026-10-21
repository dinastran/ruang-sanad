package models

type ClassMonitoringFilter struct {
	StartDate string
	EndDate   string
	GuruID    int64
	KelasID   int64
	Status    string
}

// ClassMonitoringSummary holds operational counts (all sessions) and KPI
// counts. KPI only covers official schedules whose outcome is final
// (Dinilai = terlaksana + tidak terlaksana); cancelled schedules and sessions
// still open or not yet started are reported separately and never lower it.
type ClassMonitoringSummary struct {
	Total                  int64   `json:"total"`
	BelumMulai             int64   `json:"belum_mulai"`
	Berlangsung            int64   `json:"berlangsung"`
	Selesai                int64   `json:"selesai"`
	Dibatalkan             int64   `json:"dibatalkan"`
	PerluTindakan          int64   `json:"perlu_tindakan"`
	TotalJadwal            int64   `json:"total_jadwal"`
	Dinilai                int64   `json:"dinilai"`
	Terlaksana             int64   `json:"terlaksana"`
	TerlaksanaSesuaiJadwal int64   `json:"terlaksana_sesuai_jadwal"`
	Reschedule             int64   `json:"reschedule"`
	Badal                  int64   `json:"badal"`
	TidakTerlaksana        int64   `json:"tidak_terlaksana"`
	BelumFinal             int64   `json:"belum_final"`
	JadwalDibatalkan       int64   `json:"jadwal_dibatalkan"`
	KeterlaksanaanJadwal   float64 `json:"keterlaksanaan_jadwal"`
	KetepatanJadwal        float64 `json:"ketepatan_jadwal"`
}

type MonitoringAttendance struct {
	SantriID         int64  `json:"santri_id"`
	SantriNama       string `json:"santri_nama"`
	IDMahasantri     string `json:"id_mahasantri"`
	AttendanceStatus string `json:"attendance_status"`
	AttendanceNote   string `json:"attendance_note"`
	BatasMateri      string `json:"batas_materi"`
}

type MonitoringNote struct {
	ID         int64  `json:"id"`
	Note       string `json:"note"`
	AuthorName string `json:"author_name"`
	CreatedAt  string `json:"created_at"`
}

type ScheduleActivity struct {
	ID        int64  `json:"id"`
	Action    string `json:"action"`
	Details   string `json:"details"`
	ActorName string `json:"actor_name"`
	CreatedAt string `json:"created_at"`
}

type ClassMonitoringItem struct {
	ScheduleID         int64                  `json:"schedule_id"`
	TanpaJadwal        bool                   `json:"tanpa_jadwal"`
	IsTambahan         bool                   `json:"is_tambahan"`
	KelasID            int64                  `json:"kelas_id"`
	NamaKelas          string                 `json:"nama_kelas"`
	Angkatan           string                 `json:"angkatan"`
	Level              string                 `json:"level"`
	Frekuensi          string                 `json:"frekuensi"`
	JadwalKelas        string                 `json:"jadwal_kelas"`
	MateriIndividual   bool                   `json:"materi_individual"`
	Tanggal            string                 `json:"tanggal"`
	JamMulai           string                 `json:"jam_mulai"`
	ScheduleNote       string                 `json:"schedule_note"`
	Status             string                 `json:"status"`
	ScheduleStatus     string                 `json:"schedule_status"`
	JatuhTempo         bool                   `json:"jatuh_tempo"`
	KpiStatus          string                 `json:"kpi_status"` // terlaksana/tidak_terlaksana/belum_final/dibatalkan/akan_datang/di_luar_kpi
	IsReschedule       bool                   `json:"is_reschedule"`
	JadwalSemula       string                 `json:"jadwal_semula"`
	AlasanReschedule   string                 `json:"alasan_reschedule"`
	GuruUtamaID        *int64                 `json:"guru_utama_id,omitempty"`
	GuruUtamaNama      string                 `json:"guru_utama_nama"`
	GuruPenggantiID    *int64                 `json:"guru_pengganti_id,omitempty"`
	GuruPenggantiNama  string                 `json:"guru_pengganti_nama"`
	AlasanBadal        string                 `json:"alasan_badal"`
	AlasanBatal        string                 `json:"alasan_batal"`
	IsSusulan          bool                   `json:"is_susulan"`
	AlasanSusulan      string                 `json:"alasan_susulan"`
	AssignedUserID     *int64                 `json:"assigned_user_id,omitempty"`
	PertemuanID        *int64                 `json:"pertemuan_id,omitempty"`
	ActualStartTime    string                 `json:"actual_start_time"`
	ActualEndTime      string                 `json:"actual_end_time"`
	TeacherCheckedInAt string                 `json:"teacher_checked_in_at"`
	TeacherAttendance  string                 `json:"teacher_attendance"`
	StudentAttendance  string                 `json:"student_attendance"`
	ActiveStudentCount int64                  `json:"active_student_count"`
	AttendanceCount    int64                  `json:"attendance_count"`
	Materi             string                 `json:"materi"`
	MeetingNote        string                 `json:"meeting_note"`
	NeedsAction        bool                   `json:"needs_action"`
	Attendance         []MonitoringAttendance `json:"attendance"`
	Notes              []MonitoringNote       `json:"notes"`
	Activities         []ScheduleActivity     `json:"activities"`
}

type ClassMonitoringResponse struct {
	Summary  ClassMonitoringSummary `json:"summary"`
	Items    []ClassMonitoringItem  `json:"items"`
	Dipindah []ClassMonitoringMoved `json:"dipindah"`
}

// ClassMonitoringMoved marks the original slot of a rescheduled class, so the
// koordinator sees "dipindah ke ..." on the old date. It is never scored.
type ClassMonitoringMoved struct {
	ScheduleID        int64  `json:"schedule_id"`
	KelasID           int64  `json:"kelas_id"`
	NamaKelas         string `json:"nama_kelas"`
	JadwalSemula      string `json:"jadwal_semula"`
	TanggalSemula     string `json:"tanggal_semula"`
	JamSemula         string `json:"jam_semula"`
	TanggalBaru       string `json:"tanggal_baru"`
	JamBaru           string `json:"jam_baru"`
	AlasanReschedule  string `json:"alasan_reschedule"`
	ScheduleStatus    string `json:"schedule_status"`
	GuruUtamaID       *int64 `json:"guru_utama_id,omitempty"`
	GuruUtamaNama     string `json:"guru_utama_nama"`
	GuruPenggantiID   *int64 `json:"guru_pengganti_id,omitempty"`
	GuruPenggantiNama string `json:"guru_pengganti_nama"`
}

type MonitoringReminderRequest struct {
	Kind    string `json:"kind"`
	Message string `json:"message"`
}

type MonitoringNoteRequest struct {
	Note string `json:"note"`
}

type MonitoringCancelRequest struct {
	Reason string `json:"reason"`
}

type NotificationResponse struct {
	ID        int64  `json:"id"`
	Type      string `json:"type"`
	Title     string `json:"title"`
	Message   string `json:"message"`
	ActionURL string `json:"action_url"`
	Read      bool   `json:"read"`
	CreatedAt string `json:"created_at"`
}
