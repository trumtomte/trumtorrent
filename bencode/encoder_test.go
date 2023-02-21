package bencode

import (
	"bytes"
	"testing"
)

// TODO: In general, more tests, some of which should include:
// 			- nested lists & dictionarys
// 			- encode/decode
// 			- int8, int16, int32 and int64
//
// torrent := "d8:announce33:http://explodie.org:6969/announce13:announce-listll33:http://explodie.org:6969/announceel32:http://tracker.tfile.me/announceel44:http://bigfoot1942.sektori.org:6969/announceel29:udp://eddie4.nl:6969/announceel40:udp://tracker4.piratux.com:6969/announceel40:udp://tracker.trackerfix.com:80/announceel33:udp://tracker.pomf.se:80/announceel38:udp://torrent.gresille.org:80/announceel30:udp://9.rarbg.me:2710/announceel49:udp://tracker.leechers-paradise.org:6969/announceel34:udp://glotorrents.pw:6969/announceel42:udp://tracker.opentrackr.org:1337/announceel44:udp://tracker.blackunicorn.xyz:6969/announceel48:udp://tracker.internetwarriors.net:1337/announceel34:udp://p4p.arenabg.ch:1337/announceel43:udp://tracker.coppersurfer.tk:6969/announceel30:udp://9.rarbg.to:2710/announceel44:udp://tracker.openbittorrent.com:80/announceel32:udp://explodie.org:6969/announceel44:udp://tracker.piratepublic.com:1337/announceel42:udp://tracker.aletorrenty.pl:2710/announceel41:udp://tracker.sktorrent.net:6969/announceel30:udp://zer0day.ch:1337/announceel32:udp://thetracker.org:80/announceel42:udp://tracker.pirateparty.gr:6969/announceel30:udp://9.rarbg.to:2730/announceel38:udp://bt.xxx-tracker.com:2710/announceel38:udp://tracker.cyberia.is:6969/announceel42:udp://retracker.lanta-net.ru:2710/announceel30:udp://9.rarbg.to:2770/announceel30:udp://9.rarbg.me:2730/announceel36:udp://tracker.mg64.net:6969/announceel35:udp://open.demonii.si:1337/announceel38:udp://tracker.zer0day.to:1337/announceel40:udp://tracker.tiny-vps.com:6969/announceel39:udp://ipv6.tracker.harry.lu:80/announceel30:udp://9.rarbg.me:2740/announceel30:udp://9.rarbg.me:2770/announceel42:udp://denis.stalker.upeer.me:6969/announceel39:udp://tracker.port443.xyz:6969/announceel38:udp://tracker.moeking.me:6969/announceel37:udp://exodus.desync.com:6969/announceel30:udp://9.rarbg.to:2740/announceel30:udp://9.rarbg.to:2720/announceel39:udp://tracker.justseed.it:1337/announceel41:udp://tracker.torrent.eu.org:451/announceel39:udp://ipv4.tracker.harry.lu:80/announceel44:udp://tracker.open-internet.nl:6969/announceel36:udp://torrentclub.tech:6969/announceel33:udp://open.stealth.si:80/announceel35:http://tracker.tfile.co:80/announceee7:comment61:Torrent downloaded from torrent cache at http://itorrents.org10:created by13:uTorrent/221013:creation datei1474716550e8:encoding5:UTF-84:infod6:lengthi2142264058e4:name54:The.Thing.1982.REMASTERED.1080p.BluRay.6CH.ShAaNiG.mkv12:piece lengthi2097152eee"

func TestEncodeString(t *testing.T) {
	input := "hello"
	// e := NewEncoder()
	result, err := Encode(input)

	if err != nil {
		t.Fatalf("Unable to encode string %v", input)
	}

	b := bytes.NewBuffer(result)

	if b.String() != "5:hello" {
		t.Fatalf("Encoded string %v not encoded as '5:hello'", b.String())
	}
}

func TestEncodeInteger(t *testing.T) {
	input := 42
	// e := NewEncoder()
	result, err := Encode(input)

	if err != nil {
		t.Fatalf("Unable to encode integer %v", input)
	}

	b := bytes.NewBuffer(result)

	if b.String() != "i42e" {
		t.Fatalf("Encoded integer %v not encoded as 'i42e'", b.String())
	}
}

func TestEncodeNegativeInteger(t *testing.T) {
	input := -42
	// e := NewEncoder()
	result, err := Encode(input)

	if err != nil {
		t.Fatalf("Unable to encode integer %v", input)
	}

	b := bytes.NewBuffer(result)

	if b.String() != "i-42e" {
		t.Fatalf("Encoded integer %v not encoded as 'i-42e'", b.String())
	}
}

func TestEncodeList(t *testing.T) {
	input := make([]any, 3, 3)
	input[0] = "hello"
	input[1] = 42
	input[2] = "world"
	// e := NewEncoder()
	result, err := Encode(input)

	if err != nil {
		t.Fatalf("Unable to encode list %v", input)
	}

	b := bytes.NewBuffer(result)

	if b.String() != "l5:helloi42e5:worlde" {
		t.Fatalf("Encoded list %v not encoded as 'l5:helloi42e5:worlde'", b.String())
	}
}

func TestEncodeDictionary(t *testing.T) {
	list := make([]any, 3)
	list[0] = "foo"
	list[1] = 24
	list[2] = "bar"
	input := make(map[string]any)
	input["hello"] = "world"
	input["number"] = 42
	input["values"] = list

	// e := NewEncoder()
	result, err := Encode(input)

	if err != nil {
		t.Fatalf("Unable to encode list %v", input)
	}

	b := bytes.NewBuffer(result)

	bencoded := "d5:hello5:world6:numberi42e6:valuesl3:fooi24e3:baree"

	if b.String() != bencoded {
		t.Fatalf("Encoded dict %v not encoded as '%v'", b.String(), bencoded)
	}
}
