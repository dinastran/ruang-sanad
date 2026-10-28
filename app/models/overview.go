package models

// Overview deliberately separates current snapshots from the selected month's
// activity. Nil sections mean unavailable, never a successful zero result.
type Overview struct {
	Bulan        string                       `json:"bulan"`
	BulanLabel   string                       `json:"bulan_label"`
	UpdatedAt    string                       `json:"updated_at"`
	CurrentMonth bool                         `json:"current_month"`
	Santri       *SantriAnalyticsSummary      `json:"santri,omitempty"`
	Growth       []SantriAnalyticsGrowthPoint `json:"growth"`
	Programs     []OverviewProgram            `json:"programs"`
	Operations   *OverviewOperations          `json:"operations,omitempty"`
	Riayah       *RiayahRingkasan             `json:"riayah,omitempty"`
	Attendance   []OverviewAttendanceMonth    `json:"attendance"`
	Leads        *OverviewLeads               `json:"leads,omitempty"`
	Priorities   []OverviewPriority           `json:"priorities"`
	Products     *ProductCRMDashboardSummary  `json:"products,omitempty"`
	Finance      *KeuanganDashboard           `json:"finance,omitempty"`
	Errors       []string                     `json:"errors"`
}

type OverviewProgram struct {
	Name     string `json:"name"`
	Active   int64  `json:"active"`
	Cuti     int64  `json:"cuti"`
	Stopped  int64  `json:"stopped"`
	Classes  int64  `json:"classes"`
	Seats    int64  `json:"seats"`
	Occupied int64  `json:"occupied"`
}

type OverviewOperations struct {
	Incomplete     int64 `json:"incomplete"`
	Teachers       int64 `json:"teachers"`
	Classes        int64 `json:"classes"`
	WithoutTeacher int64 `json:"without_teacher"`
	WithoutClass   int64 `json:"without_class"`
	OpenMeetings   int64 `json:"open_meetings"`
	FollowUps      int64 `json:"follow_ups"`
	Todos          int64 `json:"todos"`
	Finished       int64 `json:"finished"`
	Badal          int64 `json:"badal"`
	Rescheduled    int64 `json:"rescheduled"`
}

// OverviewAttendanceMonth counts attendance records of finished meetings,
// grouped by meeting month.
type OverviewAttendanceMonth struct {
	Month    string `json:"month"`
	Meetings int64  `json:"meetings"`
	Hadir    int64  `json:"hadir"`
	Izin     int64  `json:"izin"`
	Sakit    int64  `json:"sakit"`
	Alpa     int64  `json:"alpa"`
}

// OverviewLeads compares WhatsApp leads of the selected month with the month
// before: leads by first message date, closings by conversion date.
type OverviewLeads struct {
	Masuk             int64 `json:"masuk"`
	Closing           int64 `json:"closing"`
	MasukSebelumnya   int64 `json:"masuk_sebelumnya"`
	ClosingSebelumnya int64 `json:"closing_sebelumnya"`
}

type OverviewPriority struct {
	Title  string `json:"title"`
	Detail string `json:"detail"`
	Href   string `json:"href"`
	Count  int64  `json:"count"`
}
