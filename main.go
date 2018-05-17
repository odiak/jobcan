package main

import "net/http"
import "net/http/cookiejar"
import "net/url"
import "encoding/json"
import "os"
import "os/user"
import "fmt"
import "bytes"
import "regexp"
import "io"

type Config struct {
	CompanyID, Email, Password string
}

const (
	start  = 1
	finish = 2
)

func getOperation() (operation int, ok bool) {
	if len(os.Args) < 2 {
		return 0, false
	}
	arg1 := os.Args[1]
	switch arg1 {
	case "start":
		return start, true
	case "finish":
		return finish, true
	default:
		return 0, false
	}
}

func readConfig(config *Config) error {
	u, err := user.Current()
	if err != nil {
		return err
	}
	file, err := os.Open(u.HomeDir + "/.config/jobcan/config.json")
	if err != nil {
		return err
	}
	dec := json.NewDecoder(file)
	return dec.Decode(config)
}

func login(client http.Client, config Config) error {
	v := url.Values{}
	v.Set("client_id", config.CompanyID)
	v.Set("email", config.Email)
	v.Set("password", config.Password)
	res, err := client.PostForm("https://ssl.jobcan.jp/login/pc-employee", v)
	if err != nil {
		return err
	}
	if res.StatusCode != 200 {
		return fmt.Errorf(res.Status)
	}
	return nil
}

func bodyToString(r io.Reader) string {
	buf := new(bytes.Buffer)
	buf.ReadFrom(r)
	return string(buf.Bytes())
}

func getTokenAndGroupID(client http.Client) (token string, groupID string, err error) {
	res, err := client.Get("https://ssl.jobcan.jp/employee/")
	if err != nil {
		return
	}
	body := bodyToString(res.Body)

	token = regexp.MustCompile("name=\"token\"\\s+value=\"(.+?)\"").FindStringSubmatch(body)[1]
	groupID = regexp.MustCompile("<option\\s+value=\"(.+?)\"").FindStringSubmatch(body)[1]
	return
}

func doOperation(client http.Client, operation int, token, groupID string) error {
	var item string
	switch operation {
	case start:
		item = "work_start"
	case finish:
		item = "work_end"
	}
	v := url.Values{}
	v.Set("adit_item", item)
	v.Set("adit_group_id", groupID)
	v.Set("token", token)
	v.Set("notice", "")

	res, err := client.PostForm("https://ssl.jobcan.jp/employee/index/adit", v)
	if err != nil {
		return err
	}
	if res.StatusCode != 200 {
		return fmt.Errorf(res.Status)
	}

	fmt.Println(bodyToString(res.Body))

	return nil
}

func main() {
	operation, ok := getOperation()
	if !ok {
		os.Stderr.WriteString("Please specify operation: start or finish.")
		return
	}

	var config Config
	if err := readConfig(&config); err != nil {
		panic(err)
	}

	cli := http.Client{}
	jar, err := cookiejar.New(nil)
	if err != nil {
		panic(err)
	}
	cli.Jar = jar
	if err := login(cli, config); err != nil {
		panic(err)
	}

	token, groupID, err := getTokenAndGroupID(cli)
	if err != nil {
		panic(err)
	}

	if err := doOperation(cli, operation, token, groupID); err != nil {
		panic(err)
	}
}
