package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ashwanthkumar/slack-go-webhook"
	"github.com/mackerelio/mackerel-client-go"
)

var (
	memItem = []string{
		"memory.total",
	}
	username = os.Getenv("USERNAME")
	slackURL = os.Getenv("SLACKURL")
	mkrKey   = os.Getenv("MKRKEY")
	client   = mackerel.NewClient(mkrKey)
)

const (
	duration = 1
)

type OrgInfo struct {
	orgname string
}

type HostParams struct {
	hostIDs       []string
	hostName      string
	basicUnixTime int64
}

type HostMetricsParams struct {
	toUnixTime    int64
	fromUnixTime  int64
	memValue      float64
	errorHostList []string
}

type MemValue struct {
	Time  int64       `json:time`
	Value interface{} `json:value`
}

type OrgName struct {
	Name string `json:name`
}

func main() {
	var errorCount int
	var errorHostList []string
	var result float64

	oi := &OrgInfo{}
	oi.fetchOrgname()

	hp := &HostParams{}
	hp.FetchHostID()
	fmt.Printf("HOST-ID List: ")
	fmt.Printf("%s\n\n", hp.hostIDs)
	fmt.Println("[Fail List] \nホスト名:")
	fmt.Println("-----------")

	hmp := HostMetricsParams{}

	errorCount = 0

	for _, hostId := range hp.hostIDs {
		//result = hmp.FetchMetricsValues("3AtUb5m8gAAAAA")
		result = hmp.FetchMetricsValues(hostId)

		if result == 0 {
			hostName := fetchHostname(hostId)
			fmt.Printf("%s\n", hostName)
			errorCount += 1
			errorHostList = append(errorHostList, hostName)
		}
	}

	if errorCount >= 1 {
		failHosts := strings.Join(errorHostList, ",")
		PostSlack(oi.orgname, failHosts)
	}
}

func fetchHostname(hostId string) string {
	host, err := client.FindHost(hostId)
	if err != nil {
		fmt.Println("no hosts")
		os.Exit(0)
	}
	return host.Name
	//fmt.Printf("HOSTNAME: \t\t%s\n", hp.hostName)
}

func (oi *OrgInfo) fetchOrgname() {
	var orgName OrgName
	org, err := client.GetOrg()
	if err != nil {
		fmt.Println("Error not get orgname")
		os.Exit(0)
	}
	orgJSON, _ := json.Marshal(org)
	bytesOrg := []byte(orgJSON)

	if err := json.Unmarshal(bytesOrg, &orgName); err != nil {
		fmt.Println("JSON Unmarshal error:", err)
	}

	oi.orgname = orgName.Name
	fmt.Printf("ORG: %s\n", oi.orgname)
}

// 全host-idを習得
func (hp *HostParams) FetchHostID() {
	// Service=prodのインスタンスを取得
	basicTime := time.Now().Add(-5 * time.Minute)
	hp.basicUnixTime = basicTime.Unix()

	hostsProd, err := client.FindHosts(
		&mackerel.FindHostsParam{
			Service:  "prod",
			Roles:    []string{"web", "bastion"},
			Statuses: []string{"working"},
		},
	)
	if err != nil {
		fmt.Println("Error")
		os.Exit(0)
	}

	// Service=stgのインスタンスを取得
	hostsStg, err := client.FindHosts(
		&mackerel.FindHostsParam{
			Service:  "stg",
			Roles:    []string{"web", "bastion"},
			Statuses: []string{"working"},
		},
	)
	if err != nil {
		fmt.Println("Error")
		os.Exit(0)
	}

	// prod host-idをリストに追加
	for _, v := range hostsProd {

		// 5分以内に登録されたホストは対象外
		if v.CreatedAt < int32(hp.basicUnixTime) {
			hp.hostIDs = append(hp.hostIDs, v.ID)
		}
	}

	// stg host-idをリストに追加
	for _, v := range hostsStg {

		// 5分以内に登録されたホストは対象外
		if v.CreatedAt < int32(hp.basicUnixTime) {
			hp.hostIDs = append(hp.hostIDs, v.ID)
		}
	}
}

//func calcTotalCPUPercentPerItem(cv []CPUValue) float64 {
func convertFloat64(mv []MemValue) float64 {
	var value float64
	for i := range mv {
		value = mv[i].Value.(float64)
	}
	return value
}

func jsonFormat(m []mackerel.MetricValue, cv *[]MemValue) {
	bytesJSON, _ := json.Marshal(m)
	bytes := []byte(bytesJSON)

	if err := json.Unmarshal(bytes, &cv); err != nil {
		fmt.Println("JSON Unmarshal error:", err)
	}
}

func (hmp *HostMetricsParams) FetchMetricsValues(strHostID string) float64 {
	var memValue float64
	var metricsMemValue []MemValue
	var beforeTime = (-2 * time.Duration(duration)) - 1
	memItemsValue := [][]mackerel.MetricValue{}

	toTime := time.Now().Add(-2 * time.Minute)
	hmp.toUnixTime = toTime.Unix()

	fromTime := time.Now().Add(beforeTime * time.Minute)
	hmp.fromUnixTime = fromTime.Unix()

	for i := range memItem {
		tmp, _ := client.FetchHostMetricValues(strHostID, memItem[i], hmp.fromUnixTime, hmp.toUnixTime)
		memItemsValue = append(memItemsValue, tmp)
	}

	for i := range memItemsValue {
		jsonFormat(memItemsValue[i], &metricsMemValue)
		memValue = convertFloat64(metricsMemValue)

	}
	//fmt.Println(memValue)
	//fmt.Printf("%T", memValue[0])
	return memValue
}

func PostSlack(orgName string, failHosts string) {
	field0 := slack.Field{Title: "ORGNAME", Value: orgName}
	field1 := slack.Field{Title: "Mackerel 自動退役失敗 or Agent停止", Value: failHosts}

	attachment := slack.Attachment{}
	attachment.AddField(field0).AddField(field1)
	color := "warning"
	attachment.Color = &color
	payload := slack.Payload{
		Username: username,
		//Channel:     channel,
		Attachments: []slack.Attachment{attachment},
	}
	err := slack.Send(slackURL, "", payload)
	if len(err) > 0 {
		os.Exit(1)
	}
}
