package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

var (
	requestUrl     = "https://www.jisilu.cn/data/cbnew/pre_list/"
	headers = map[string]string{
		"Origin":           "https://www.jisilu.cn",
		"User-Agent":       "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_2) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/79.0.3945.88 Safari/537.36",
		"X-Requested-With": "XMLHttpRequest",
		"Referer":          "https://www.jisilu.cn/data/cbnew/",
		"Accept":           "application/json, text/javascript, */*; q=0.01",
		"Content-Type":     "application/x-www-form-urlencoded; charset=UTF-8",
		"Accept-Language":  "en-US,en;q=0.9,zh-CN;q=0.8,zh;q=0.7",
	}
	requestData = "{\"cb_type_Y\": \"Y\", \"progress\": \"\", \"rp\": 22}"
)

type CbResponse struct {
	Page int   `json:"page"`
	Rows []Row `json:"rows"`
}
type Row struct {
	Id   string `json:"id"`
	Cell Cell   `json:"cell"`
}

type Cell map[string]interface{}

func main() {
	client := http.Client{}
	request, err := http.NewRequest("POST", requestUrl, strings.NewReader(requestData))
	if err != nil {
		panic(err)
	}
	for k, v := range headers {
		request.Header.Set(k, v)
	}
	resp, err := client.Do(request)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	var data CbResponse
	err = json.Unmarshal(bytes, &data)
	if err != nil {
		panic(err)
	}
	applyCb, listedCb := getCbInfo(&data)
	msg := formatInfo(applyCb, listedCb)

	// Send msg to wx
	SendMsgByServerChan(msg, len(applyCb) > 0 )
}

func getCbInfo(data *CbResponse) (applyCb []Cell, listedCb []Cell) {
	today := time.Now().Format("2006-01-02")
	fmt.Println("Today is ", today)
	for _, row := range data.Rows {
		// 申购债券
		if row.Cell["apply_date"] == today {
			applyCb = append(applyCb, row.Cell)
		}
		// 上市债券
		if row.Cell["list_date"] == today {
			listedCb = append(listedCb, row.Cell)
		}
	}
	return
}

func formatInfo(applyList []Cell, listedList []Cell)  (msg string) {
	msg = "日期：" + time.Now().Format("2006-01-02") + "\n\n"
	if len(applyList) != 0 {
		msg += "#### *当日可打新债*：  \n"
		for _, cell := range applyList {
			msg += formatCell(cell)
		}
	} else {
		msg += "#### *当日无可打新债*  \n"
	}
	if len(listedList) != 0 {
		msg += "#### *当日上市新债*：  \n"
		for _, cell := range listedList {
			msg += formatCell(cell)
		}
	} else {
		msg += "\n#### *当日无上市新债*  \n"
	}
	msg += "\n#### 以上数据来源于互联网，仅供参考，不作为投资建议 "
	return msg
}

func formatCell(cell Cell) (str string){
	var lucky_draw_rt string
	if cell["lucky_draw_rt"] != nil {
		lucky_draw_rt = cell["lucky_draw_rt"].(string) + "%"
	} else {
		lucky_draw_rt = "---"
	}
	return fmt.Sprintf(cellTemplate,
		cell["stock_nm"],
		cell["bond_id"],
		cell["stock_id"],
		cell["price"],
		lucky_draw_rt,
		cell["rating_cd"],
		cell["jsl_advise_text"])
}

var (
	cellTemplate = `
"名称"：%s  
"债券代码": %s  
"证券代码": %s  
"现    价": %s  
"中签率": %s  
"评   级": %s  
"申购建议": %s  
`

)


var (
	serverChanUrl = "https://sc.ftqq.com/%s.send"
)

func SendMsgByServerChan(msg string, newApply bool) {

	env, b := os.LookupEnv("SERVERCHANSECRET")

	if !b {
		fmt.Println("未配置server酱 secret 跳过推送")
		return
	}

	title := ""
	if newApply {
		title = "有可转债申购"
	} else {
		title = "当日无申购"
	}

	values := url.Values{"text": {title}, "desp": {msg}}

	_, err := http.PostForm(fmt.Sprintf(serverChanUrl, env), values)

	if err != nil {
		panic(err)
	}

}
