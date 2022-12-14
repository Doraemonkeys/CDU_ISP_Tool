package server

import (
	"ISP_Tool/model"
	"ISP_Tool/utils"
	"ISP_Tool/view"
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
)

func InitConfig() error {
	log.Println("正在初始化配置文件")
	fmt.Println("正在初始化配置文件")
	err := os.MkdirAll("./config", 0666)
	if err != nil {
		log.Println("创建程序配置文件夹失败！", err)
		fmt.Println("创建程序配置文件夹失败！", err)
	}
	//检查自启动
	model.Auto_Start = CheckAutoStart()
	if model.Auto_Start {
		if TodayCheckInSuccess() {
			//已经设置为自启动并且今日打卡已成功
			log.Println("打卡程序重新运行的原因: 用户手动打开")
			fmt.Println()
			model.Auto_Clock_IN_Success = true
			view.Auto_Clock_IN_Success()
			fmt.Println()
			startTime := time.Now()
			fmt.Println()
			fmt.Printf("按Enter键继续执行程序......")
			ch := make(chan bool, 1)
			go utils.PressToContinue(ch)
			ok := false
			for !ok {
				select {
				case ok = <-ch:
				default:
					time.Sleep(time.Second / 4)
				}
				//无操作30秒退出
				if time.Since(startTime) > time.Minute/2 {
					os.Exit(0)
				}
			}
		}
	}
	fmt.Println("正在检查网络环境...")
	for i := 0; !utils.NetWorkStatus(); i++ {
		time.Sleep(time.Second)
		//最多检查10次
		if i == 10 {
			color.Red("网络连接错误，请检查网络配置!")
			log.Println("网络连接错误，请检查网络配置!")
			return errors.New("网络连接错误")
		}
	}
	fmt.Println("Net Status , OK!")
	//从网络获取全局配置
	err = GetConfig()
	if err != nil {
		log.Println("从网络获取程序配置文件失败！", err)
		fmt.Println("从网络获取程序配置文件失败！", err)
		return err
	}
	fmt.Println("从网络获取全局配置成功！")
	log.Println("从网络获取全局配置成功！")
	fmt.Println()
	fmt.Println()
	view.Menu()
	fmt.Println()
	config, err := os.Open("./config/配置文件.config")
	if err == nil {
		defer config.Close()
		temp := make([]byte, 20)
		n, err := config.Read(temp)
		if err != nil && err != io.EOF {
			log.Println("预读取配置文件失败，Error:", err)
			fmt.Println("预读取配置文件失败，Error:", err)
			return err
		}
		if n < 10 {
			log.Println("配置文件为空!")
			fmt.Println("配置文件为空!")
		} else {
			return nil
		}
	} else {
		log.Println("配置文件不存在或打开失败", err)
		fmt.Println("配置文件不存在或打开失败")
	}
	err = firstUse()
	if err != nil {
		return err
	}
	fmt.Println()
	fmt.Println()
	model.UserConfigChanged = true
	return nil
}

func RebuildConfig(users []model.UserInfo) error {
	config, err := os.OpenFile("./config/配置文件.config", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0666)
	if err != nil {
		log.Println("打开配置文件失败，Error:", err)
		fmt.Println("打开配置文件失败，Error:", err)
		return err
	}
	defer config.Close()
	for _, v := range users {
		data, err := json.Marshal(v)
		if err != nil {
			log.Println("个人信息序列化失败！", err)
			fmt.Println("个人信息序列化失败！", err)
			return err
		}
		data = append(data, '\n')
		config.Write(data)
	}
	return nil
}

// 从网络获取全局配置
func GetConfig() error {
	content, err := utils.Fetch("https://gitee.com/doraemonkey/json_isp/raw/master/json2.txt")
	if err != nil {
		return err
	}
	err = json.Unmarshal(content, &model.All)
	if err != nil {
		return err
	}
	return nil
}

func SetAutoStart() error {
	// C:\Users\*\AppData\Roaming\Microsoft\Windows\Start Menu\Programs\Startup
	// 获取当前Windows用户的home directory.
	winUserHomeDir, err := os.UserHomeDir()
	if err != nil {
		log.Println("获取当前Windows用户的用户名失败，Error:", err)
		return err
	}
	startFile := winUserHomeDir + `\AppData\Roaming\Microsoft\Windows\Start Menu\Programs\Startup` + `\isp_auto_start.vbs`
	file, err := os.OpenFile(startFile, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0666)
	if err != nil {
		log.Println("创建或打开文件失败!", err)
		return err
	}
	defer file.Close()
	path, err := os.Getwd()
	if err != nil {
		log.Println("获取当前文件目录失败！", err)
		return err
	}
	path = strings.Replace(path, `\`, `\\`, -1)
	_, err = file.WriteString(utils.Utf8ToANSI(`Set objShell = CreateObject("WScript.Shell")` + "\n"))
	if err != nil {
		log.Println("写入当前文件目录失败！", err)
		return err
	}
	_, err = file.WriteString(utils.Utf8ToANSI(`objShell.CurrentDirectory = "` + path + `\\config` + `"` + "\n"))
	if err != nil {
		log.Println("写入当前文件目录失败！", err)
		return err
	}
	_, err = file.WriteString(utils.Utf8ToANSI(`objShell.Run "powershell /c ` + ".\\*.exe" + `"` + `,0`))
	if err != nil {
		log.Println("写入当前文件目录失败！", err)
		return err
	}
	start_config, err := os.OpenFile("./config/auto_start.config", os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		log.Println("创建或打开自启动配置文件失败!", err)
		return err
	}
	defer start_config.Close()
	n, err := start_config.WriteAt([]byte("true "), 0)
	if err != nil || n != 5 {
		log.Println("写入自启动配置文件失败！", err)
		return err
	}
	err = StartNewProgram(startFile)
	if err != nil {
		return err
	}
	return nil
}

// 用户设置自启动后会关闭当前程序，延迟几秒开启一个守护进程，
// 应当确保在设置自启动后调用。
// startPath为自启动脚本的路径+文件名。
// 主体思路是：在当前目录下创建bat文件，bat文件中延迟几秒调用vbs脚本。
func StartNewProgram(startFile string) error {
	//获取程序路径
	path, err := utils.GetExecutionPath()
	if err != nil {
		log.Println("获取当前路径失败！", err)
		return fmt.Errorf("获取当前路径失败,%w", err)
	}
	//获取文件的绝对路径
	absPath, err := filepath.Abs(path)
	if err != nil {
		log.Println("获取当前路径失败！", err)
		return fmt.Errorf("获取当前路径失败,%w", err)
	}
	absPath = filepath.Dir(absPath)
	//fmt.Println(absPath) //调试用
	batFile := `startVBS.bat`
	//命令1
	cmd1 := "cmd /c " + `"` + startFile + `"`
	//命令2
	cmd2 := "del " + batFile
	f, err := os.Create(absPath + `\` + batFile)
	if err != nil {
		log.Println("创建批处理文件失败！", err)
		return fmt.Errorf("创建批处理文件失败,%w", err)
	}
	_, err = f.WriteString(`if "%1" == "h" goto begin
	mshta vbscript:createobject("wscript.shell").run("""%~nx0"" h",0)(window.close)&&exit
	:begin` + "\n")
	if err != nil {
		log.Println("写入批处理文件失败！", err)
		return fmt.Errorf("写入批处理文件失败,%w", err)
	}
	f.WriteString("ping -n 3 127.1>nul" + " & " + cmd1 + " & " + cmd2)
	f.Close()
	cmdStr := "cmd /c " + `".\` + batFile + `"`
	//执行批处理文件
	err = utils.CmdNoOutput(absPath, []string{cmdStr, "&", "exit"})
	if err != nil {
		log.Println("执行cmd命令失败！", err)
		return fmt.Errorf("执行cmd命令失败,%w", err)
	}
	return nil
}

func CancelAutoStart() error {
	// C:\Users\*\AppData\Roaming\Microsoft\Windows\Start Menu\Programs\Startup
	// 获取当前Windows用户的home directory.
	winUserHomeDir, err := os.UserHomeDir()
	if err != nil {
		log.Println("获取当前Windows用户的用户名失败，Error:", err)
		return err
	}
	startFile := winUserHomeDir + `\AppData\Roaming\Microsoft\Windows\Start Menu\Programs\Startup` + `\isp_auto_start.vbs`
	err = os.Remove(startFile)
	if err != nil {
		log.Println("删除自启动脚本失败！", err)
		return err
	}
	start_config, err := os.OpenFile("./config/auto_start.config", os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		log.Println("创建或打开自启动配置文件失败!", err)
		return err
	}
	defer start_config.Close()
	n, err := start_config.WriteAt([]byte("false"), 0)
	if err != nil || n != 5 {
		log.Println("写入自启动配置文件失败！", err)
		return err
	}
	return nil
}

// 添加用户信息
func AddUser() error {
	config, err := os.OpenFile("./config/配置文件.config", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		log.Println("打开配置文件失败，Error:", err)
		fmt.Println("打开配置文件失败，Error:", err)
		return err
	}
	defer config.Close()
	var users []model.UserInfo
	var NewUser = model.UserInfo{}
	for {
		fmt.Println()
		attributes := [5]color.Attribute{}
		attributes[1] = color.FgRed
		utils.ColorPrint(attributes[:], "输入 ", "Q", " 退出添加账号\n")
		//fmt.Println("输入 Q 退出添加账号")
		fmt.Println("请输入学号：")
		var id string
		fmt.Scan(&id)
		NewUser.UserID = strings.TrimSpace(id)
		if NewUser.UserID == "Q" || NewUser.UserID == "q" {
			break
		}
		fmt.Println("请输入密码：")
		var pwd string
		fmt.Scan(&pwd)
		NewUser.UserPwd = strings.TrimSpace(pwd)
		if NewUser.UserPwd == "Q" || NewUser.UserPwd == "q" {
			break
		}
		fmt.Println("请输入教务系统密码：")
		var VPNPwd string
		fmt.Scan(&VPNPwd)
		NewUser.VPN_Pwd = strings.TrimSpace(VPNPwd)
		if NewUser.VPN_Pwd == "Q" || NewUser.VPN_Pwd == "q" {
			break
		}
		NewUser.ChooseLocation = 1
		users = append(users, NewUser)
	}
	fmt.Scanf("\n")
	for _, v := range users {
		data, err := json.Marshal(v)
		if err != nil {
			log.Println("个人信息序列化失败！", err)
			fmt.Println("个人信息序列化失败！", err)
			return err
		}
		data = append(data, '\n')
		config.Write(data)
		fmt.Printf("添加 %s 成功！\n", v.UserID)
	}
	return nil
}

func SwitchChooseLocation() error {
	var targetUser model.UserInfo
	attributes := [5]color.Attribute{}
	attributes[2] = color.FgRed
	utils.ColorPrint(attributes[:], "请输入需要切换的学号：", "(输入", "ALL", "更改全部用户)\n")
	var id string
	fmt.Scan(&id)
	attributes2 := [10]color.Attribute{}
	attributes2[2] = color.FgRed
	attributes2[6] = color.FgRed
	utils.ColorPrint(attributes2[:], "请选择打卡地址获取方式: ", "[", "1", "]", "IP地址 ", "[", "2", "]", "ISP历史打卡地址\n")
	var targetWay string
	fmt.Scan(&targetWay)
	targetUser.UserID = strings.ToUpper(strings.TrimSpace(id))
	targetWay = strings.TrimSpace(targetWay)
	if targetWay != "1" && targetWay != "2" {
		return errors.New("选择打卡地址获取方式错误！")
	}
	target, _ := strconv.Atoi(targetWay)
	config, err := os.Open("./config/配置文件.config")
	if err != nil {
		log.Println("打开配置文件失败！", err)
		fmt.Println("打开配置文件失败！", err)
		return err
	}
	defer config.Close()
	reader := bufio.NewReader(config)
	users := []model.UserInfo{}
	var user model.UserInfo
	found := false
	for {
		userData, err := reader.ReadString('\n')
		if err == io.EOF {
			if len(userData) > 1 {
				userData = strings.TrimSpace(userData)
				json.Unmarshal([]byte(userData), &user)
				if user.UserID == targetUser.UserID || targetUser.UserID == "ALL" {
					user.ChooseLocation = target
					found = true
				}
				users = append(users, user)
			}
			if !found && targetUser.UserID != "ALL" {
				return errors.New("没有找到目标ID")
			}
			break
		}
		if err != nil {
			log.Println("读取配置文件失败！", err)
			fmt.Println("读取配置文件失败！", err)
			return err
		}
		userData = strings.TrimSpace(userData)
		json.Unmarshal([]byte(userData), &user)
		if user.UserID == targetUser.UserID || targetUser.UserID == "ALL" {
			user.ChooseLocation = target
			found = true
		}
		users = append(users, user)
	}
	err = RebuildConfig(users)
	if err != nil {
		log.Println("修改配置文件失败！", err)
		fmt.Println("修改配置文件失败！", err)
		return err
	}
	return nil
}
