package steam

func GetWorkShopItemInfos(ids []string) ([]*WorkShopItem, error) {
	//log.DebugF("get workshop item info: %v", ids)
	//
	//c := util.ProxyClient().SetTimeout(5 * time.Second)
	//
	//var w sync.WaitGroup
	//w.Add(len(ids))
	//
	//var wl []*WorkShopItem
	//var lock sync.Mutex

	//for _, id := range ids {
	//	go func(id string) {
	//		defer w.Done()
	//
	//		resp, err := c.R().Get(fmt.Sprintf("https://steamcommunity.com/sharedfiles/filedetails/?id=%s", id))
	//		if err != nil {
	//			log.ErrorF("connect failed: %s", err.NewErr(err))
	//			return
	//		}
	//		if resp.StatusCode() != http.StatusOK {
	//			log.ErrorF("request failed: %s", err.NewErr(resp))
	//			return
	//		}
	//
	//		doc, err := htmlquery.Parse(strings.NewReader(resp.String()))
	//		if err != nil {
	//			log.ErrorF("parse html failed: %s", err.NewErr(err))
	//			return
	//		}
	//		res := htmlquery.FindOne(doc, "//div[@class='workshopItemTitle']")
	//		if res == nil {
	//			log.ErrorF("xpath failed: %s", id)
	//			return
	//		}
	//
	//		i := WorkShopItem{
	//			Id:   id,
	//			Name: htmlquery.InnerText(res),
	//		}
	//		lock.Lock()
	//		defer lock.Unlock()
	//		wl = append(wl, &i)
	//	}(id)
	//}

	//w.Wait()
	//return wl, nil

	return nil, nil
}
