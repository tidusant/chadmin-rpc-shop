package main

import (
	"flag"
	"fmt"
	"net"

	"github.com/tidusant/c3m-common/c3mcommon"
	"github.com/tidusant/c3m-common/inflect"
	"github.com/tidusant/c3m-common/log"

	rpch "github.com/tidusant/chadmin-repo/cuahang"
	"github.com/tidusant/chadmin-repo/models"

	"encoding/json"
	"net/rpc"
	"strconv"
	"strings"
)

const (
	defaultcampaigncode string = "XVsdAZGVmY"
)

type ConfigViewData struct {
	ShopConfigs     models.ShopConfigs
	TemplateConfigs []ConfigItem
	BuildConfigs    models.BuildConfig
}
type ConfigItem struct {
	Key   string
	Type  string
	Value string
}

type Arith int

func (t *Arith) Run(data string, result *models.RequestResult) error {
	*result = c3mcommon.ReturnJsonMessage("0", "no action found", "", `{}`)
	//parse  args
	log.Debugf("call with " + data)
	args := strings.Split(data, "|")

	if len(args) < 3 {
		return nil
	}
	var usex models.UserSession
	usex.Session = args[0]
	usex.Action = args[2]
	info := strings.Split(args[1], "[+]")
	if len(info) < 2 {
		return nil
	}
	usex.UserID = info[0]
	ShopID := info[1]
	usex.Params = ""
	if len(args) > 3 {
		usex.Params = args[3]
	}
	//check shop permission
	shop := rpch.GetShopById(usex.UserID, ShopID)
	if shop.Status == 0 {
		*result = c3mcommon.ReturnJsonMessage("-4", "Shop is disabled.", "", "")
		return nil
	}
	usex.Shop = shop

	if usex.Action == "cs" {
		*result = ChangeShop(usex)

	} else if usex.Action == "lsi" {
		*result = loadshopinfo(usex)
	} else if usex.Action == "ca" {
		*result = doCreateAlbum(usex)
	} else if usex.Action == "la" {
		*result = doLoadalbum(usex)
	} else if usex.Action == "ea" {
		*result = doEditAlbum(usex)
	} else if usex.Action == "cga" {
		*result = configGetAll(usex)
	} else if usex.Action == "cgs" {
		*result = configSave(usex)
	} else if usex.Action == "lims" {
		*result = getShopLimits(usex)
	}

	return nil
}

// func savecat(usex models.UserSession) string {
// 	log.Debugf("createcat data: %s", params)
// 	var catinf models.ShopCat
// 	err := json.Unmarshal([]byte(usex.Params), &catinf)
// 	if !c3mcommon.CheckError("createcat parse json", err) {
// 		return c3mcommon.ReturnJsonMessage("0", "create cat fail", "", "")
// 	} test dev

// 	code := rpch.SaveCat(usex.UserID, usex.ShopID, catinf)

// 	if code == "-1" {
// 		return c3mcommon.ReturnJsonMessage("2", "max cat limited", "", "")
// 	} else if code != "" {
// 		return c3mcommon.ReturnJsonMessage("1", "", "success", """+code+""")
// 	}

// 	return c3mcommon.ReturnJsonMessage("0", "create cat fail", "", "")
// }

//func savecat(usex models.UserSession) string {
//	args := strings.Split(usex.Params, ",")
//	if len(args) < 3 {
//		return c3mcommon.ReturnJsonMessage("0", "error submit fields", "", "")
//	}
//	name := args[0]
//	desc := args[1]
//	code := args[2]
//	lang := args[3]
//	catinfo := models.ShopCatInfo{Name: name, Description: desc, Lang: lang}
//	rt := rpch.SaveCat(usex.UserID, usex.ShopID, code, catinfo)
//	if rt == "1" {
//		return c3mcommon.ReturnJsonMessage("1", "", "success", "")
//	}

//	return c3mcommon.ReturnJsonMessage("0", "update cat fail", "", "")
//}
func loadshopinfo(usex models.UserSession) models.RequestResult {
	strrt := `{"Shop":`
	b, _ := json.Marshal(usex.Shop)
	strrt += string(b)

	//get langs info
	strrt += `,"Languages":[`
	for _, lang := range usex.Shop.Config.Langs {
		strrt += `{"Code":"` + lang + `","Name":"` + c3mcommon.GetLangnameByCode(lang) + `","Flag":"` + c3mcommon.Code2Flag(lang) + `"},`
	}
	if len(usex.Shop.Config.Langs) > 0 {
		strrt = strrt[:len(strrt)-1] + `]`
	}
	b, _ = json.Marshal(usex.Shop.Config)
	strrt += `,"ShopConfigs":` + string(b)

	//maxfileupload
	strrt += `,"MaxFileUpload":` + strconv.Itoa(rpch.GetShopLimitbyKey(usex.Shop.ID.Hex(), "maxfileupload"))
	strrt += `,"MaxSizeUpload":` + strconv.Itoa(rpch.GetShopLimitbyKey(usex.Shop.ID.Hex(), "maxsizeupload"))

	//orther shop
	otherShops := rpch.GetOtherShopById(usex.UserID, usex.Shop.ID.Hex())
	strrt += `,"Others":[`
	for _, shop := range otherShops {
		strrt += `{"Name":"` + shop.Name + `","ID":"` + shop.ID.Hex() + `"},`
	}
	if len(otherShops) > 0 {
		strrt = strrt[:len(strrt)-1] + `]`
	} else {
		strrt += `]`
	}

	//get user info
	user := rpch.GetUserInfo(usex.UserID)
	strrt += `,"User":{"Name":"` + user.Name + `"}`
	strrt += "}"

	log.Debugf("loadshopinfo: " + strrt)

	return c3mcommon.ReturnJsonMessage("1", "", "success", strrt)

}

func ChangeShop(usex models.UserSession) models.RequestResult {
	shop := rpch.GetShopById(usex.UserID, usex.Params)
	// if shop.Status == 0 {
	// 	return c3mcommon.ReturnJsonMessage("-4", "Shop is disabled.", "", "")
	// }
	//update login session

	if !rpch.UpdateShopLogin(usex.Session, shop.ID.Hex()) {
		return c3mcommon.ReturnJsonMessage("-4", "Change shop fail.", "", "")
	}

	return c3mcommon.ReturnJsonMessage("1", "", "success", "")

}
func configSave(usex models.UserSession) models.RequestResult {
	var config ConfigViewData

	err := json.Unmarshal([]byte(usex.Params), &config)
	if !c3mcommon.CheckError("json parse page", err) {
		return c3mcommon.ReturnJsonMessage("0", "json parse fail", "", "")
	}
	usex.Shop.Config = config.ShopConfigs
	rpch.SaveShopConfig(usex.Shop)

	// //save template config
	str := `{"Code":"` + usex.Shop.Theme + `","TemplateConfigs":[{}`
	for _, conf := range config.TemplateConfigs {
		str += `,{"Key":"` + conf.Key + `","Value":"` + conf.Value + `"}`
	}
	str += `]`
	b, _ := json.Marshal(config.BuildConfigs)
	str += `,"BuildConfig":` + string(b) + `}`

	request := "savetemplateconfig|" + usex.Session
	resp := c3mcommon.RequestBuildService(request, "POST", str)

	if resp.Status != "1" {
		return resp
	}

	// //save build config

	// var bcf models.BuildConfig
	// bcf = config.BuildConfigs
	// bcf.ShopId = usex.Shop.ID.Hex()
	// rpb.SaveConfig(bcf)
	//rebuild config
	rpch.Rebuild(usex)
	return c3mcommon.ReturnJsonMessage("1", "", "success", "")

}
func configGetAll(usex models.UserSession) models.RequestResult {
	var config ConfigViewData
	config.ShopConfigs = usex.Shop.Config
	log.Debugf("configGetAll")
	request := "gettemplateconfig|" + usex.Session
	resp := c3mcommon.RequestBuildService(request, "POST", usex.Shop.Theme)
	log.Debugf("RequestBuildService call done")
	if resp.Status != "1" {
		return resp
	}
	var confs struct {
		TemplateConfigs []ConfigItem
		BuildConfigs    models.BuildConfig
	}
	json.Unmarshal([]byte(resp.Data), &confs)

	config.TemplateConfigs = confs.TemplateConfigs
	config.BuildConfigs = confs.BuildConfigs
	config.BuildConfigs.ID = ""
	config.BuildConfigs.ShopId = ""
	b, _ := json.Marshal(config)

	return c3mcommon.ReturnJsonMessage("1", "", "success", string(b))

}
func getShopLimits(usex models.UserSession) models.RequestResult {

	limits := rpch.GetShopLimits(usex.Shop.ID.Hex())

	b, _ := json.Marshal(limits)
	return c3mcommon.ReturnJsonMessage("1", "", "success", string(b))

}

// func loadcat(usex models.UserSession) string {
// 	log.Debugf("loadcat begin")
// 	shop := rpch.GetShopById(usex.UserID, usex.ShopID)

// 	strrt := "["
// 	log.Debugf("load cats:%v", shop.ShopCats)
// 	catinfstr := ""
// 	for _, cat := range shop.ShopCats {
// 		catlangs := ""
// 		for lang, catinf := range cat.Langs {
// 			catlangs += """ + lang + "":{"name":"" + catinf.Slug + "","slug":"" + catinf.Name + "","description":"" + catinf.Description + ""},"
// 		}
// 		catlangs = catlangs[:len(catlangs)-1]
// 		catinfstr += "{"code":"" + cat.Code + "","langs":{" + catlangs + "}},"
// 	}
// 	if catinfstr == "" {
// 		strrt += "{}]"
// 	} else {
// 		strrt += catinfstr[:len(catinfstr)-1] + "]"
// 	}

// 	return c3mcommon.ReturnJsonMessage("1", "", "success", strrt)

// }

func doCreateAlbum(usex models.UserSession) models.RequestResult {
	albumname := usex.Params
	if albumname == "" {
		return c3mcommon.ReturnJsonMessage("0", "albumname empty", "", "")
	}
	//get config

	if usex.Shop.ID.Hex() == "" {
		return c3mcommon.ReturnJsonMessage("0", "shop not found", "", "")
	}

	// if usex.Shop.Config.Level == 0 {
	// 	return c3mcommon.ReturnJsonMessage("0", "config error", "", "")

	// }
	// if usex.Shop.Config.MaxAlbum <= len(usex.Shop.Albums) {
	// 	return c3mcommon.ReturnJsonMessage("2", "album count limited", "", "")
	// }

	slug := strings.ToLower(inflect.Camelize(inflect.Asciify(albumname)))
	albumslug := slug
	if slug == "" {
		return c3mcommon.ReturnJsonMessage("0", "albumslug empty", "", "")
	}

	//save albumname
	var album models.ShopAlbum
	album.Slug = albumslug
	album.Name = albumname
	album.ShopID = usex.Shop.ID.Hex()
	album.UserId = usex.UserID
	album = rpch.SaveAlbum(album)
	b, _ := json.Marshal(album)

	return c3mcommon.ReturnJsonMessage("1", "", "success", string(b))

}
func doLoadalbum(usex models.UserSession) models.RequestResult {

	//get albums
	albums := rpch.LoadAllShopAlbums(usex.Shop.ID.Hex())
	if len(albums) == 0 {
		//create
		var album models.ShopAlbum
		album.Slug = "default"
		album.Name = "Default"
		album.ShopID = usex.Shop.ID.Hex()
		album.UserId = usex.UserID
		album = rpch.SaveAlbum(album)
		albums = append(albums, album)
	}

	b, err := json.Marshal(albums)
	c3mcommon.CheckError("json parse doLoadalbum", err)
	return c3mcommon.ReturnJsonMessage("1", "", "", string(b))

}
func doEditAlbum(usex models.UserSession) models.RequestResult {
	//log.Debugf("update album ")
	var newitem models.ShopAlbum
	log.Debugf("Unmarshal %s", usex.Params)
	err := json.Unmarshal([]byte(usex.Params), &newitem)
	if !c3mcommon.CheckError("json parse page", err) {
		return c3mcommon.ReturnJsonMessage("0", "json parse ShopAlbum fail", "", "")
	}
	newitem.ShopID = usex.Shop.ID.Hex()
	newitem.UserId = usex.UserID
	rpch.SaveAlbum(newitem)
	//log.Debugf("update album false %s", albumname)
	return c3mcommon.ReturnJsonMessage("0", "album not found", "", "")

}
func main() {
	var port int
	var debug bool
	flag.IntVar(&port, "port", 9879, "help message for flagname")
	flag.BoolVar(&debug, "debug", false, "Indicates if debug messages should be printed in log files")
	flag.Parse()

	logLevel := log.DebugLevel
	if !debug {
		logLevel = log.InfoLevel

	}

	log.SetOutputFile(fmt.Sprintf("adminShop-"+strconv.Itoa(port)), logLevel)
	defer log.CloseOutputFile()
	log.RedirectStdOut()

	//init db
	arith := new(Arith)
	rpc.Register(arith)
	log.Infof("running with port:" + strconv.Itoa(port))

	//			rpc.HandleHTTP()
	//			l, e := net.Listen("tcp", ":"+strconv.Itoa(port))
	//			if e != nil {
	//				log.Debug("listen error:", e)
	//			}
	//			http.Serve(l, nil)

	tcpAddr, err := net.ResolveTCPAddr("tcp", ":"+strconv.Itoa(port))
	c3mcommon.CheckError("rpc dail:", err)

	listener, err := net.ListenTCP("tcp", tcpAddr)
	c3mcommon.CheckError("rpc init listen", err)

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go rpc.ServeConn(conn)
	}
}
