package login

import (
	"ISP_Tool/model"
	"ISP_Tool/utils"
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"

	"golang.org/x/text/transform"
)

// 登陆客户端

func LoginISP(client *http.Client, user model.UserInfo) error {
	code, err := Get_ISP_Login_code(client)
	if err != nil {
		return err
	}
	fmt.Println("成功获取登录验证码:", code)
	log.Println("成功获取登录验证码:", code)
	data := "username=" + user.UserID
	data = data + "&userpwd=" + user.UserPwd
	data = data + "&code=" + code
	data = data + "&login=login&checkcode=1&rank=0&action=login&m5=1"
	request, _ := http.NewRequest("POST",
		"https://xsswzx.cdu.edu.cn/ispstu/com_user/weblogin.asp", strings.NewReader(data))

	request.Header.Set("authority", "xsswzx.cdu.edu.cn")
	request.Header.Set("content-type", "application/x-www-form-urlencoded")
	request.Header.Set("user-agent", model.UserAgent)
	request.Header.Set("referer", "https://xsswzx.cdu.edu.cn/ispstu/com_user/login.asp")
	// 发起登录请求
	resp, err := client.Do(request)
	if err != nil {
		log.Println("发起ISP登录请求失败！可能是ISP结构发生变化，请联系开发者。")
		fmt.Println("发起ISP登录请求失败！可能是ISP结构发生变化，请联系开发者。")
		return err
	}
	bodyReader := bufio.NewReader(resp.Body)
	//自动检测html编码
	e, err := utils.DetermineEncodingbyPeek(bodyReader)
	if err != nil {
		log.Println("登录返回界面检测html编失败，请联系开发者。", err)
		fmt.Println("登录返回界面检测html编失败，请联系开发者。", err)
		return err
	}
	//转码utf-8
	utf8BodyReader := transform.NewReader(bodyReader, e.NewDecoder())

	content, err := ioutil.ReadAll(utf8BodyReader)
	if err != nil {
		log.Println("读取登录返回界面失败！", err)
		fmt.Println("读取登录返回界面失败！", err)
		return err
	}
	re := regexp.MustCompile("密码错误")
	match := re.FindSubmatch(content)
	if match != nil {
		log.Println("账号或者密码错误，请修改。", "账号：", user.UserID, "密码：", user.UserPwd)
		fmt.Println("账号或者密码错误，请修改。", "账号：", user.UserID, "密码：", user.UserPwd)
		return errors.New("账号或者密码错误")
	}
	re = regexp.MustCompile("非在校学生")
	match = re.FindSubmatch(content)
	if match != nil {
		log.Println("账号或者密码错误，请修改。", "账号：", user.UserID, "密码：", user.UserPwd)
		fmt.Println("账号或者密码错误，请修改。", "账号：", user.UserID, "密码：", user.UserPwd)
		return errors.New("账号或者密码错误")
	}
	return nil
}
