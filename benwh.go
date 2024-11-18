// Package benwh reads status values from the FranklinWH servers for a
// specified device using the credentials provided by the client.
package benwh

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// LoginResult holds the user details for an authenticated user.
type LoginResult struct {
	UserID  int    `json:"userId"`
	Email   string `json:"email"`
	Token   string `json:"token"`
	Version string `json:"version"`
	// "failureVersion":null
	// "distributorId":null
	// "installerId":null
	PasswordUpdateFlag  int   `json:"passwordUpdateFlag"`
	UserTypes           []int `json:"userTypes"`
	CurrentType         int   `json:"currentType"`
	SurveyFlag          int   `json:"surveyFlag"`
	ServiceVoltageFlag  int   `json:"serviceVoltageFlag"`
	NinetyDaysPwdUpdate int   `json:"ninetyDaysPwdUpdate"`
}

// LoginResponse is what an attempt to obtain a login token returns.
type LoginResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Result  LoginResult `json:"result"`
	Total   int         `json:"total"`
	Success bool        `json:"success"`
}

// MQTTRequest is the structure used to make MQTT requests of the
// service.
type MQTTRequest struct {
	Lang      string `json:"lang"`
	CmdType   int    `json:"cmdType"`
	EquipNo   string `json:"equipNo"`
	Type      int    `json:"type"`
	TimeStamp int64  `json:"timeStamp"`
	Snno      int    `json:"snno"`
	// Len and CRC are for the marshaled form of the dataArea
	// request.
	Len int    `json:"len"`
	CRC string `json:"crc"`
	// DataArea is post-encoding replaced without quotes in json output.
	DataArea string `json:"dataArea"`
}

// MQTTResult is the returned .Result data after making a MQTTRequest.
type MQTTResult struct {
	CmdType   int    `json:"cmdType"`
	EquipNo   string `json:"equipNo"`
	Type      int    `json:"type"`
	TimeStamp int64  `json:"timeStamp"`
	Snno      int    `json:"snno"`
	Len       int    `json:"len"`
	CRC       string `json:"crc"`
	DataArea  string `json:"dataArea"`
}

// MQTTResponse is the response to a MQTTRequest sent to the server.
type MQTTResponse struct {
	Code    int        `json:"code"`
	Message string     `json:"message"`
	Result  MQTTResult `json:"result"`
	Total   int        `json:"total"`
	Success bool       `json:"success"`
}

// DataStatus is the unmarshaled dataArea value from a successful
// status request.
type DataStatus struct {
	ReportType        int       `json:"report_type"`
	Mode              int       `json:"mode"`
	RunStatus         int       `json:"run_status"`
	SlaverStat        int       `json:"slaver_stat"`
	ElecnetState      int       `json:"elecnet_state"`
	FhpSn             []string  `json:"fhpSn"`
	InfiStatus        []int     `json:"infi_status"`
	PeStat            []int     `json:"pe_stat"`
	BmsWork           []int     `json:"bms_work"`
	BmsHeatState      []int     `json:"bms_heat_state"`
	PUti              float64   `json:"p_uti"`
	PSun              float64   `json:"p_sun"`
	PGen              float64   `json:"p_gen"`
	PFhp              float64   `json:"p_fhp"`
	PLoad             float64   `json:"p_load"`
	KwhUtiIn          float64   `json:"kwh_uti_in"`
	KwhUtiOut         float64   `json:"kwh_uti_out"`
	KwhSun            float64   `json:"kwh_sun"`
	KwhGen            float64   `json:"kwh_gen"`
	KwhFhpDi          float64   `json:"kwh_fhp_di"`
	KwhFhpChg         float64   `json:"kwh_fhp_chg"`
	KwhLoad           float64   `json:"kwh_load"`
	SolarSupply       float64   `json:"solarSupply"`
	Soc               float64   `json:"soc"`
	FhpSoc            []float64 `json:"fhpSoc"`
	FhpPower          []float64 `json:"fhpPower"`
	TAmb              float64   `json:"t_amb"`
	MainSw            []int     `json:"main_sw"`
	ProLoad           []int     `json:"pro_load"`
	Do                []int     `json:"do"`
	Di                []int     `json:"di"`
	Signal            int       `json:"signal"`
	WifiSignal        int       `json:"wifiSignal"`
	ConnType          int       `json:"connType"`
	GenStat           int       `json:"genStat"`
	KwhSolarLoad      float64   `json:"kwhSolarLoad"`
	KwhGridLoad       float64   `json:"kwhGridLoad"`
	KwhFhpLoad        float64   `json:"kwhFhpLoad"`
	KwhGenLoad        float64   `json:"kwhGenLoad"`
	RemoteSolarEn     int       `json:"remoteSolarEn"`
	SolarPower        int       `json:"solarPower"`
	RemoteSolar1Power int       `json:"remoteSolar1Power"`
	RemoteSolar2Power int       `json:"remoteSolar2Power"`
	DSPNEMPVPower     int       `json:"DSPNEMPVPower"`
	BFPVApboxRelay    int       `json:"BFPVApboxRelay"`
	BatOutGrid        int       `json:"batOutGrid"`
	SoOutGrid         int       `json:"soOutGrid"`
	GridChBat         int       `json:"gridChBat"`
	GenChBat          int       `json:"genChBat"`
	SinHTemp          int       `json:"sinHTemp"`
	SinLTemp          int       `json:"sinLTemp"`
	Name              string    `json:"name"`
	Electricity_type  int       `json:"electricity_type"`
	GridPhaseConSet   int       `json:"gridPhaseConSet"`
}

const urlBase = "https://energy.franklinwh.com/"

// URLBase is the base URL for service requests associated with
// FranklinWH devices.
var URLBase = urlBase

// Config holds client authentication and access information.
type Config struct {
	Email    string
	Device   []string
	Password string
}

// Conn holds an open connection to the service.
type Conn struct {
	config Config
	token  string
	client *http.Client
}

// NewConn creates an authenticated connection to ther FranklinWH
// service.
func NewConn(conf Config) (conn *Conn, err error) {
	c := &http.Client{}
	v := url.Values{}
	v.Set("account", conf.Email)
	v.Set("password", conf.Password)
	v.Set("lang", "EN_US")
	v.Set("type", "1")
	v.Set("user-agent", userAgent)
	res, err := c.PostForm(URLBase+"hes-gateway/terminal/initialize/appUserOrInstallerLogin", v)
	if err != nil {
		return nil, err
	}
	d, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	res.Body.Close()
	var resp LoginResponse
	if err := json.Unmarshal(d, &resp); err != nil {
		return nil, err
	}
	conn = &Conn{
		config: conf,
		token:  resp.Result.Token,
		client: c,
	}
	return
}

// ErrRetryLater is used to indicate a timeout occurred and retrying after
// some wait time is likely to work.
var ErrRetryLater = errors.New("retry later")

// Status returns the system status.
func (conn *Conn) Status() (resp *DataStatus, err error) {
	req := []byte(`{"opt":1,"refreshData":1}`)
	checksum := crc32.ChecksumIEEE(req)

	send := MQTTRequest{
		Lang:      "EN_US",
		CmdType:   203,
		EquipNo:   conn.config.Device[0],
		Type:      0,
		TimeStamp: time.Now().Unix(),
		Snno:      1,
		Len:       len(req),
		CRC:       fmt.Sprintf("%08X", checksum),
		DataArea:  ":data-area:",
	}
	j, err2 := json.Marshal(send)
	if err2 != nil {
		err = fmt.Errorf("preparation failed: %v", err2)
		return
	}
	j = bytes.Replace(j, []byte(`":data-area:"`), []byte(req), 1)

	fReq, err2 := http.NewRequest("POST", URLBase+"hes-gateway/terminal/sendMqtt", bytes.NewBuffer(j))
	if err != nil {
		err = fmt.Errorf("query preparation failed: %v", err2)
		return
	}
	fReq.Header.Add("loginToken", conn.token)
	fReq.Header.Add("Content-Type", "application/json")
	fReq.Header.Add("user-agent", userAgent)

	res, err2 := conn.client.Do(fReq)
	if err2 != nil {
		err = fmt.Errorf("query failed: %v", err2)
		return
	}

	d, err2 := io.ReadAll(res.Body)
	if err2 != nil {
		err = fmt.Errorf("failed to read body: %v", err2)
		return
	}

	var mresp MQTTResponse
	if err2 := json.Unmarshal(d, &mresp); err != nil {
		err = fmt.Errorf("mqtt response decode error: %v", err2)
		return
	}
	switch mresp.Code {
	case 102:
		// Seems to require a simple retry.
		err = ErrRetryLater
		return
	case 200:
		// OK
	default:
		err = fmt.Errorf("unexpected mqtt code = %d", mresp.Code)
		return
	}

	checksum = crc32.ChecksumIEEE([]byte(fmt.Sprintf("%q", mresp.Result.DataArea)))
	if got, err2 := strconv.ParseUint(mresp.Result.CRC, 16, 32); err != nil {
		err = fmt.Errorf("invalid CRC return got=%q which is not hex", mresp.Result.CRC, err2)
		return
	} else if uint32(got) != checksum {
		err = fmt.Errorf("invalid CRC return got=%X, want=%X", got, checksum)
		return
	}
	resp = &DataStatus{}
	if err2 := json.Unmarshal([]byte(mresp.Result.DataArea), resp); err != nil {
		err = fmt.Errorf("status report decode error: %v", err2)
		return
	}
	return
}
