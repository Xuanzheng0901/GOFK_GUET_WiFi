package main

import (
	"GoProject0/internal/winapi/wlan"
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/go-resty/resty/v2"
	"golang.org/x/term"
)

type UserInfo struct {
	UserName string `json:"username"`
	Pwd      string `json:"pwd"`
}

func connect() {
	client := wlan.NewWlanHandle()
	client.Init()
	if client.SSID != "GUET-WiFi" {
		log.Println("未正确连接,正在断开已有网络")
		wlan.Disconnect()
		for {
			time.Sleep(1 * time.Second)
			_, state := client.CheckConnetionState()
			if !state {
				break
			}
		}

		log.Println("正在连接")
		wlan.ConnectBySSID("GUET-WiFi")
		waitCount := 0
		for {
			ssid, state := client.CheckConnetionState()
			time.Sleep(1 * time.Second)
			if state {
				log.Printf("已连接到WiFi:%s", ssid)
				break
			}
			waitCount++
			if waitCount > 10 {
				log.Fatalln("连接超时,请手动连接校园网")
			}
		}
	}
	client.Close()
}

func main() {
	fmt.Print("提示: 若连接过程中弹出浏览器界面,直接关闭浏览器即可\n\n")
	wd, err := os.Getwd()
	if err != nil {
		fmt.Println("无法获取当前工作目录:", err)
	}
	log.Println("程序当前的工作目录是:", wd)
	connect()

	time.Sleep(1 * time.Second)

	infoFile := "info.json"
	var userName, pwd string
	_, err = os.Stat(infoFile)
	if err == nil {
		data, _ := os.ReadFile(infoFile)
		var info UserInfo
		_ = json.Unmarshal(data, &info)
		userName, pwd = info.UserName, info.Pwd
		log.Println("正在使用已保存的账户密码登录...")
	} else {
		scanner := bufio.NewScanner(os.Stdin)
		fmt.Print("首次使用,请输入智慧校园账号/学号:")
		for {
			if scanner.Scan() {
				userName = scanner.Text()
				if len(userName) != 10 {
					fmt.Print("输入格式不正确!请重新输入:")
				} else {
					break
				}
			}
		}

		fmt.Print("请输入密码: ")
		bytePwd, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			_ = scanner.Scan()
			pwd = scanner.Text()
		} else {
			pwd = string(bytePwd)
		}
		pwd = base64.StdEncoding.EncodeToString([]byte(pwd))
	}

	mac := wlan.GetLocalMAC()
	url := fmt.Sprintf("http://10.0.1.5:801/eportal/portal/login?user_account=%%2C0%%2C%s&user_password=%s&wlan_user_mac=%s&wlan_ac_name=HJ-BRAS-ME60-01",
		userName, pwd, mac)

	client := resty.New()
	client.SetTimeout(3 * time.Second)

	resp, err := client.R().
		SetHeaders(map[string]string{
			"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/127.0.0.0 Safari/537.36 Edg/127.0.0.0",
			"Referer":    "https://10.0.1.5:801/",
			"Host":       "10.0.1.5",
		}).
		Get(url)

	if err != nil {
		log.Fatalln("请求失败:", err)
		return
	}
	re := regexp.MustCompile(`\{.*}`)
	match := re.Find(resp.Body())
	if match == nil {
		log.Fatalln("请求失败:", err)
		return
	}
	var result map[string]interface{}
	if err := json.Unmarshal(match, &result); err == nil {
		//if true {
		msg, _ := result["msg"].(string)
		fmt.Println(string(resp.Body()))
		//fmt.Println(resp.StatusCode(), msg)

		if !strings.Contains(msg, "ldap auth error") && resp.StatusCode() == 200 {
			// 保存账户信息
			//if true {
			_, err := os.Stat(infoFile)
			if os.IsNotExist(err) {
				data, _ := json.MarshalIndent(UserInfo{UserName: userName, Pwd: pwd}, "", "    ")
				_ = os.WriteFile(infoFile, data, 0644)
				log.Println("认证成功,已将账户密码加密存入程序目录下的info.json文件,下次认证无需输入")
			}
		} else if strings.Contains(msg, "ldap auth error") {
			_ = os.Remove(infoFile)
		}
	}
}
