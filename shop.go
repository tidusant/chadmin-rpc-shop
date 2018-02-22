package main

import (
	"flag"
	"net"
	"time"

	"github.com/tidusant/c3m-common/c3mcommon"
	"github.com/tidusant/c3m-common/inflect"
	"github.com/tidusant/c3m-common/log"
	rpch "github.com/tidusant/chadmin-repo/cuahang"
	"github.com/tidusant/chadmin-repo/models"
	rpimg "github.com/tidusant/chadmin-repo/vrsgim"

	"net/rpc"
	"strconv"
	"strings"
)

const (
	defaultcampaigncode string = "XVsdAZGVmY"
)

type Arith int

func (t *Arith) Run(data string, result *string) error {
	log.Debugf("Call RPCshop args:" + data)
	*result = ""
	//parse args
	args := strings.Split(data, "|")

	if len(args) < 3 {
		return nil
	}
	var usex models.UserSession
	usex.Session = args[0]
	usex.Action = args[2]
	info := strings.Split(args[1], "[+]")
	usex.UserID = info[0]
	ShopID := info[1]
	usex.Params = ""
	if len(args) > 3 {
		usex.Params = args[3]
	}
	//check shop permission
	shop := rpch.GetShopById(usex.UserID, ShopID)
	if shop.Status == 0 {
		//load shop by default
		shopid := rpch.GetShopDefault(usex.UserID)
		if shopid == "" {
			*result = c3mcommon.ReturnJsonMessage("-4", "Shop is disabled.", "", "")
			return nil
		}
		shop = rpch.LoadShopById(usex.Session, usex.UserID, shopid)

	}
	usex.Shop = shop
	if usex.Action == "cd" {
		*result = checkdomain(usex)
	} else if usex.Action == "cs" {
		*result = ChangeShop(usex)

	} else if usex.Action == "c" {
		*result = createshop(usex)

	} else if usex.Action == "lsi" {
		*result = loadshopinfo(usex)
	} else if usex.Action == "lco" {
		*result = loadshopconfig(usex)
	} else if usex.Action == "ca" {
		*result = doCreateAlbum(usex)
	} else if usex.Action == "la" {
		*result = doLoadalbum(usex)
	} else if usex.Action == "ea" {
		*result = doEditAlbum(usex)
	} else { //default
		*result = ""
	}

	return nil
}

// func savecat(usex models.UserSession) string {
// 	log.Debugf("createcat data: %s", params)
// 	var catinf models.ShopCat
// 	err := json.Unmarshal([]byte(usex.Params), &catinf)
// 	if !c3mcommon.CheckError("createcat parse json", err) {
// 		return c3mcommon.ReturnJsonMessage("0", "create cat fail", "", "")
// 	}

// 	code := rpch.SaveCat(usex.UserID, usex.ShopID, catinf)

// 	if code == "-1" {
// 		return c3mcommon.ReturnJsonMessage("2", "max cat limited", "", "")
// 	} else if code != "" {
// 		return c3mcommon.ReturnJsonMessage("1", "", "success", "\""+code+"\"")
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
func loadshopinfo(usex models.UserSession) string {

	flag := "{"
	if usex.Shop.Config.Multilang {
		for _, lang := range usex.Shop.Config.Langs {
			flag += "\"" + lang + "\":{\"lang\":\"" + lang + "\",\"flag\":\"" + c3mcommon.Code2Flag(lang) + "\"},"
		}
		flag = flag[:len(flag)-1] + "}"
	}
	log.Debugf("shopinfo:%s", flag)

	strrt := "{\"Theme\":\"" + usex.Shop.Theme + "\",\"currentlang\":\"" + usex.Shop.Config.CurrentLang + "\",\"langs\":" + flag

	//get config

	tempconf := rpch.GetTemplateConfigs(usex.Shop.ID.Hex(), usex.Shop.Theme)
	if len(tempconf) > 0 {
		confstr := "{"
		for _, confg := range tempconf {
			confstr += `"` + confg.Key + `":"` + confg.Value + `",`
		}
		confstr = confstr[:len(confstr)-1] + "}"
		strrt += `,"Configs":` + confstr
	}
	//system config

	strrt += `,"SysConfigs":{"Shopid":"` + usex.Shop.ID.Hex() + `","Name":"` + usex.Shop.Name + `","Phone":"` + usex.Shop.Phone + `","Avatar":"` + usex.Shop.Config.Avatar + `","GHTKToken":"` + usex.Shop.Config.GHTKToken + `","FBPageId":"` + usex.Shop.Config.FBPageId + `","ShipFee":` + strconv.Itoa(usex.Shop.Config.ShipFee) + `,"FreeShip":` + strconv.Itoa(usex.Shop.Config.FreeShip) + `}`

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

	strrt += "}"

	return c3mcommon.ReturnJsonMessage("1", "", "success", strrt)

}

func ChangeShop(usex models.UserSession) string {
	shop := rpch.GetShopById(usex.UserID, usex.Params)
	if shop.Status == 0 {
		return c3mcommon.ReturnJsonMessage("-4", "Shop is disabled.", "", "")
	}
	//update login session

	if !rpch.UpdateShopLogin(usex.Session, shop.ID.Hex()) {
		return c3mcommon.ReturnJsonMessage("-4", "Change shop fail.", "", "")
	}

	return c3mcommon.ReturnJsonMessage("1", "", "success", "")

}

func loadshopconfig(usex models.UserSession) string {

	userdomain := "0"
	if usex.Shop.Config.Userdomain {
		userdomain = "1"
	}
	ftpdomain := usex.Shop.Config.Domain
	ftpuser := usex.Shop.Config.Ftpusername
	filecount := rpimg.ImageCount(usex.Shop.ID.Hex())
	prodcatscount := len(rpch.GetAllCats(usex.UserID, usex.Shop.ID.Hex()))
	strrt := "{\"name\":\"" + usex.Shop.Name + "\",\"domain\":\"" + usex.Shop.Domain + "\",\"userdomain\":\"" + userdomain + "\",\"ftpdomain\":\"" + ftpdomain + "\",\"ftpuser\":\"" + ftpuser + "\",\"cats\":\"" + strconv.Itoa(prodcatscount) + "/" + strconv.Itoa(usex.Shop.Config.MaxCat) + "\",\"users\":\"" + strconv.Itoa(len(usex.Shop.Users)) + "/" + strconv.Itoa(usex.Shop.Config.MaxUser) + "\",\"albums\":\"" + strconv.Itoa(len(usex.Shop.Albums)) + "/" + strconv.Itoa(usex.Shop.Config.MaxAlbum) + "\",\"images\":\"" + strconv.Itoa(filecount) + "/" + strconv.Itoa(usex.Shop.Config.MaxImage) + "\"}"

	return c3mcommon.ReturnJsonMessage("1", "", "success", strrt)

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
// 			catlangs += "\"" + lang + "\":{\"name\":\"" + catinf.Slug + "\",\"slug\":\"" + catinf.Name + "\",\"description\":\"" + catinf.Description + "\"},"
// 		}
// 		catlangs = catlangs[:len(catlangs)-1]
// 		catinfstr += "{\"code\":\"" + cat.Code + "\",\"langs\":{" + catlangs + "}},"
// 	}
// 	if catinfstr == "" {
// 		strrt += "{}]"
// 	} else {
// 		strrt += catinfstr[:len(catinfstr)-1] + "]"
// 	}

// 	return c3mcommon.ReturnJsonMessage("1", "", "success", strrt)

// }
func createshop(usex models.UserSession) string {
	// colshop := db.C("addons_shops")
	// args := strings.Split(usex.Params, ",")
	// if len(args) < 2 {
	// 	return ""
	// }
	// shopname := args[0]
	// domainname := args[1]
	// //check domain
	// count, err := colshop.Find(bson.M{"domain": domainname}).Count()
	// c3mcommon.CheckError("checkdomain", err)
	// if count == 0 {
	// 	var newshop models.Shop
	// 	newshop.Name = shopname
	// 	newshop.Domain = domainname
	// 	//newshop.Id = bson.ObjectIdHex(usex.UserID)
	// 	newshop.Created = time.Now().UTC().Add(7 * time.Hour)
	// 	err := colshop.Insert(newshop)
	// 	c3mcommon.CheckError("Upsert login", err)
	// 	return "1"
	// }

	return ""
}

func doCreateAlbum(usex models.UserSession) string {
	albumname := usex.Params
	if albumname == "" {
		return c3mcommon.ReturnJsonMessage("0", "albumname empty", "", "")
	}
	//get config

	if usex.Shop.ID.Hex() == "" {
		return c3mcommon.ReturnJsonMessage("0", "shop not found", "", "")
	}

	if usex.Shop.Config.Level == 0 {
		return c3mcommon.ReturnJsonMessage("0", "config error", "", "")

	}
	if usex.Shop.Config.MaxAlbum <= len(usex.Shop.Albums) {
		return c3mcommon.ReturnJsonMessage("2", "album count limited", "", "")
	}

	slug := strings.ToLower(inflect.Camelize(inflect.Asciify(albumname)))
	albumslug := slug
	if slug == "" {
		return c3mcommon.ReturnJsonMessage("0", "albumslug empty", "", "")
	}
	i := 1
	//get array of album slug
	albumslugs := map[string]string{}
	for _, album := range usex.Shop.Albums {
		albumslugs[album.Slug] = album.Name

	}
	for {
		if _, ok := albumslugs[albumslug]; ok {
			albumslug = slug + strconv.Itoa(i)
			i++
		} else {
			break
		}
	}

	//save albumname
	usex.Shop.Albums = append(usex.Shop.Albums, models.ShopAlbum{Slug: albumslug, Name: albumname, Created: time.Now().UTC().Add(7 * time.Hour)})
	rpch.UpdateAlbum(usex.Shop)

	return c3mcommon.ReturnJsonMessage("1", "", "success", "\""+albumslug+"\"")

}
func doLoadalbum(usex models.UserSession) string {

	//get config

	strrt := "{\"\":\"\""
	for _, album := range usex.Shop.Albums {
		strrt += ",\"" + album.Slug + "\":\"" + album.Name + "\""
	}

	strrt += "}"
	return c3mcommon.ReturnJsonMessage("1", "", "", strrt)

}
func doEditAlbum(usex models.UserSession) string {
	//log.Debugf("update album ")
	params := strings.Split(usex.Params, ",")
	if len(params) < 2 {
		return c3mcommon.ReturnJsonMessage("0", "albumname edit error", "", "")
	}
	albumslug := params[0]
	albumname := params[1]
	if albumname == "" {
		return c3mcommon.ReturnJsonMessage("0", "albumname empty", "", "")
	}
	//get config

	for i, album := range usex.Shop.Albums {
		if album.Slug == albumslug {
			usex.Shop.Albums[i].Name = albumname
			rpch.UpdateAlbum(usex.Shop)
			//log.Debugf("update album %s", albumname)
			return c3mcommon.ReturnJsonMessage("1", "", "success", "\""+albumname+"\"")
		}
	}
	//log.Debugf("update album false %s", albumname)
	return c3mcommon.ReturnJsonMessage("0", "album not found", "", "")

}
func checkdomain(usex models.UserSession) string {

	// colshop := db.C("addons_shops")

	// domainname := params
	// i := 0

	// for {
	// 	domainname = params
	// 	if i > 0 {
	// 		domainname = domainname + strconv.Itoa(i)
	// 	}
	// 	count, err := colshop.Find(bson.M{"domain": domainname}).Count()
	// 	c3mcommon.CheckError("checkdomain", err)
	// 	log.Debugf("found domain %s %d", domainname, count)
	// 	if count == 0 {
	// 		break
	// 	}

	// 	i++
	// }
	// return domainname
	return ""
}

func main() {
	var port int
	var debug bool
	flag.IntVar(&port, "port", 9879, "help message for flagname")
	flag.BoolVar(&debug, "debug", false, "Indicates if debug messages should be printed in log files")
	flag.Parse()

	// logLevel := log.DebugLevel
	// if !debug {
	// 	logLevel = log.InfoLevel

	// }

	// log.SetOutputFile(fmt.Sprintf("adminShop-"+strconv.Itoa(port)), logLevel)
	// defer log.CloseOutputFile()
	// log.RedirectStdOut()

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
