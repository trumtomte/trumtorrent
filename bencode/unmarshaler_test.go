package bencode

import (
	"testing"
)

// TODO: Add more test, eg.
// 		- Test nested slice of ints/strings
// 		- Test map[string]string/int
// 		- Test field as Struct
// 		- Test slice of Structs
// NOTE: save temporarily incase we need to do more testing
// Tested with:
// s := "d5:firsti42e6:second5:hello5:thirdli40ei41ee6:fourthl3:foo3:bare5:fifthll5:helloel5:worldee5:sixthd3:foo3:bare7:seventhd7:numbersli42ei41eee5:infosld3:foo3:abced3:foo3:defeee"
//
// type Info struct {
// 	Foo string `bencode:"foo"`
// }
//
// type Demo struct {
// First   int              `bencode:"first"`
// Second  string           `bencode:"second"`
// Third   []int            `bencode:"third"`
// Fourth  []string         `bencode:"fourth"`
// Fifth   [][]string       `bencode:"fifth"`
// Sixth   Info             `bencode:"sixth"`
// Seventh map[string][]int `bencode:"seventh"`
// Infos   []Info           `bencode:"infos"`
// Sixth   map[string]string `bencode:"sixth"`
// }

func TestUnmarshalStructWithIntAndStr(t *testing.T) {
	type Container struct {
		A int    `bencode:"a"`
		B string `bencode:"b"`
	}

	data := make(map[string]any)
	data["a"] = 42
	data["b"] = "Hello"

	d, _ := Encode(data)
	c := &Container{}
	err := Unmarshal(d, c)

	if err != nil {
		t.Fatalf("Unable to unmarshal %v into %T", data, c)
	}

	if c.A != 42 {
		t.Fatalf("Unable to unmarshal 'int' from '%s' into %T", data, c)
	}

	if c.B != "Hello" {
		t.Fatalf("Unable to unmarshal 'string' from '%s' into %T", data, c)
	}
}

func TestUnmarshalStructWithSliceOfIntAndStr(t *testing.T) {
	type Container struct {
		A []int    `bencode:"a"`
		B []string `bencode:"b"`
	}

	a := make([]any, 2, 2)
	a[0] = 42
	a[1] = 41
	b := make([]any, 2, 2)
	b[0] = "Hello"
	b[1] = "World"

	data := make(map[string]any)
	data["a"] = a
	data["b"] = b

	d, _ := Encode(data)
	c := &Container{}
	err := Unmarshal(d, c)

	if err != nil {
		t.Fatalf("Unable to unmarshal %v into %T", data, c)
	}

	if c.A[0] != 42 {
		t.Fatalf("Unable to unmarshal '[]int' from '%s' into %T", data, c)
	}

	if c.B[0] != "Hello" {
		t.Fatalf("Unable to unmarshal '[]string' from '%s' into %T", data, c)
	}
}

// func testBencodedFile() {
// 	b, _ := os.ReadFile("the_thing.torrent")

// 	result, err := bencode.Decode(b)

// 	if err != nil {
// 		fmt.Println(err)
// 		return
// 	}

// 	metainfo := &torrent.Metainfo{}
// 	err = bencode.Unmarshal(result, metainfo)

// 	fmt.Println(metainfo.Announce)
// 	fmt.Println(metainfo.Info.PieceLength)
// 	fmt.Println(metainfo.Info.Name)
// }

// func testBencodedString() {
// 	bencodedData := "d8:announce33:http://explodie.org:6969/announce13:announce-listll33:http://explodie.org:6969/announceel32:http://tracker.tfile.me/announceel44:http://bigfoot1942.sektori.org:6969/announceel29:udp://eddie4.nl:6969/announceel40:udp://tracker4.piratux.com:6969/announceel40:udp://tracker.trackerfix.com:80/announceel33:udp://tracker.pomf.se:80/announceel38:udp://torrent.gresille.org:80/announceel30:udp://9.rarbg.me:2710/announceel49:udp://tracker.leechers-paradise.org:6969/announceel34:udp://glotorrents.pw:6969/announceel42:udp://tracker.opentrackr.org:1337/announceel44:udp://tracker.blackunicorn.xyz:6969/announceel48:udp://tracker.internetwarriors.net:1337/announceel34:udp://p4p.arenabg.ch:1337/announceel43:udp://tracker.coppersurfer.tk:6969/announceel30:udp://9.rarbg.to:2710/announceel44:udp://tracker.openbittorrent.com:80/announceel32:udp://explodie.org:6969/announceel44:udp://tracker.piratepublic.com:1337/announceel42:udp://tracker.aletorrenty.pl:2710/announceel41:udp://tracker.sktorrent.net:6969/announceel30:udp://zer0day.ch:1337/announceel32:udp://thetracker.org:80/announceel42:udp://tracker.pirateparty.gr:6969/announceel30:udp://9.rarbg.to:2730/announceel38:udp://bt.xxx-tracker.com:2710/announceel38:udp://tracker.cyberia.is:6969/announceel42:udp://retracker.lanta-net.ru:2710/announceel30:udp://9.rarbg.to:2770/announceel30:udp://9.rarbg.me:2730/announceel36:udp://tracker.mg64.net:6969/announceel35:udp://open.demonii.si:1337/announceel38:udp://tracker.zer0day.to:1337/announceel40:udp://tracker.tiny-vps.com:6969/announceel39:udp://ipv6.tracker.harry.lu:80/announceel30:udp://9.rarbg.me:2740/announceel30:udp://9.rarbg.me:2770/announceel42:udp://denis.stalker.upeer.me:6969/announceel39:udp://tracker.port443.xyz:6969/announceel38:udp://tracker.moeking.me:6969/announceel37:udp://exodus.desync.com:6969/announceel30:udp://9.rarbg.to:2740/announceel30:udp://9.rarbg.to:2720/announceel39:udp://tracker.justseed.it:1337/announceel41:udp://tracker.torrent.eu.org:451/announceel39:udp://ipv4.tracker.harry.lu:80/announceel44:udp://tracker.open-internet.nl:6969/announceel36:udp://torrentclub.tech:6969/announceel33:udp://open.stealth.si:80/announceel35:http://tracker.tfile.co:80/announceee7:comment61:Torrent downloaded from torrent cache at http://itorrents.org10:created by13:uTorrent/221013:creation datei1474716550e8:encoding5:UTF-84:infod6:lengthi2142264058e4:name54:The.Thing.1982.REMASTERED.1080p.BluRay.6CH.ShAaNiG.mkv12:piece lengthi2097152eee"
// 	result, err := bencode.Decode([]byte(bencodedData))
// 	fmt.Println(err, result)

// 	// type Info struct {
// 	// 	Name        string `bencode:"name"`
// 	// 	PieceLength int    `bencode:"piece length"`
// 	// 	Length      int    `bencode:"length"`
// 	// }

// 	// type Demo struct {
// 	// 	Announce     string     `bencode:"announce"`
// 	// 	Comment      string     `bencode:"comment"`
// 	// 	CreatedBy    string     `bencode:"created by"`
// 	// 	CreationDate int        `bencode:"creation date"`
// 	// 	Encoding     string     `bencode:"encoding"`
// 	// 	AnnounceList [][]string `bencode:"announce-list"`
// 	// 	Info         Info       `bencode:"info"`
// 	// }

// 	demo := &torrent.Metainfo{}
// 	err = bencode.Unmarshal(result, demo)

// 	if err != nil {
// 		fmt.Println(err)
// 		return
// 	}

// 	fmt.Println(demo.Announce)
// 	fmt.Println(demo.Info.PieceLength)
// 	fmt.Println(demo.Info.Name)
// }
