package main

import (
	"github.com/golang/snappy"
	"encoding/base64"
	"log"
	"mime"
	"net/http"
	"path"
	"strings"
	"time"
)

const (
	cMedia_Logo36Png = "-RVMiVBORw0KGgoAAAANSUhEUgAAACQBBPD9CAYAAADhAJiYAAAAAXNSR0IArs4c6QAAAAlwSFlzAAALEwAACxMBAJqcGAAABCJpVFh0WE1MOmNvbS5hZG9iZS54bXAAAAAAADx4OnhtcG1ldGEgeG1sbnM6eD0iYWRvYmU6bnM6bWV0YS8iIHg6eG1wdGs9IlhNUCBDb3JlIDUuNC4wIj4KICAgPHJkZjpSREYgeG1sbnM6cmRmPSJodHRwOi8vd3d3LnczLm9yZy8xOTk5LzAyLzIyLXJkZi1zeW50YXgtbnMjIj4KICAgICAgPHJkZjpEZXNjcmlwdGlvbiByZGY6YWJvdXQ9IiIKICAgICAgICAgICAgeG0BqAh0aWYZaghucy4J1wxjb20vARoQLzEuMC9ONwAIZXhpWjcAARpiNwAUZGM9Imh0BdYMcHVybAXUOGRjL2VsZW1lbnRzLzEuMVJvAAh4bXAVOTKlAAh4YXAJbTEAIUsBukQ6UmVzb2x1dGlvblVuaXQ-MjwF0T4XAAAKATEFAQk2MENvbXByZXNzaW9uPjUNMy4UAD4wAABYGWcEPjcRZC4VAD4xABhPcmllbnRhAZgEPjENYS4UAD4wAABZTmEALhUAHTEheyw6UGl4ZWxYRGltZW4FxggzNjwllEIZAD45ACRDb2xvclNwYWNlAZkFTB0TPi4ABWcAWV5nAB0ZHTkkZGM6c3ViamVjdB0WIbEccmRmOkJhZy8dFwQ8L1IuAJA8eG1wOk1vZGlmeURhdGU-MjAxNi0wNi0xMFQxMDowNjo5MzwvOiQAHYIBGixDcmVhdG9yVG9vbD4FxgBtAQ8MIDMuNQk6LiAADTsEPC8BogBEeVYFGQkWGFJERj4KPC-VAPSNBj4KX-GF8AAABnBJREFUWAnNWAlQlVUU_t57LAIqIBK7ibmgTGaokQ2mGZO7Nq1OpFCaS-m4TQqM4ziZW9mY1ozlaIPbpGNpKKSJY4iKggq4oKAGriCiD9nhsfzdcx7__34e7_lEyTwz77_nnnvOvefes9xzH_CMgaaFPlHRAwGNXwt6awmSVo_Ny1JaK2anCExd4gyDIQOS1EuhNSE9vDyg02qRU1hsPvSQvnRSDA5ihimxXti4vOghzMqQVsEMtRGWlOni4YbclfOQvXwOfNw6KOytQupwAp_EhD6KjEkhwEMWCO0WgL1zJqFfFx_08X0OGo0GWvEL9vNCL-_O2D8_CuNDesvstluN5AsJRxAZO8kWs8lkKs64z95DkI8n3uzzAtROFj97InM5O9ijh1dnxGdcUknZQCU4AtJmoVRfBDoswJIljZYk1CekjK89mMo4LewkfjJQn34E83ckyuRWttJ85NcmYupCV0uCLRRytLNDxvXbaJQk5r9bVoHY3w5i0e9JKC6vZFpDYyMKH5TDwU5nac5HoY2AQZeGSTE9zZmbmSysZ1ckffkp2tmbyKTMppTTLHe_ogrrI8dzxKUt_hz1DY0IW_Yz0vJums9ru0_RrEU6oqInIG7lAVnAtLKgeLR3aqYMMRU8KJN5UVhqwolop9Oik5CxCQ6NXjA4qd3RJKKtajB1ANOZ9xscdrVIH3708jXkFesxNKgb8_UUUZV1oxABnVzxzYcjOfQlYc4YcXKr_kzBwQtX1POpcM1dDBwcj5DB7SFp7aBtsPzTaO0RHOqCc6nsDyato2KiRWiuoBlf9PdG1tJZHOqqFRSUFBq6ciNScvMV2hMjGkcnxC2paeHUNPEvk9-1qgyNU176bsIoQtscLCpEZiutrkHETzuxYOd-ZdGYXX8xraLGgPWH6WZoe2jm1PL0835NRPSuAzDUN6BvgDf7Do39kXGR77MdaeeUtCDLtFVrUSGanJQhOH-rCLO27eN-7p17TJNzFHfa-GNVIXkdcuAfD52Qu_95a9GH5FUDPd1ltM1aW9m9WR4Sq4bLK7_TPxipi6YjXWThq0X3ZbJIns7YJyoBH7eOOH7lOp4X5UlK7FSMe7k381KuoiohPLg7Fo8fhulvvIJqQx2bfvWEkUiYG8kRnJxjljI0dsuQlVxv1WRUkBEUlJQryhBCClAVMCDQD9-KxFglFqPKgJx_W6o_Jr8-QOG_U1oOb9cOoHJm95lssTE9K-Pi6KDwmCNWTUbXgiWorqtjck5BMUcaXbjX7pUwLeN6gSKyLikVkzft5n6myPQ1dfXQV1Zxv327x1BIpzUl8d6-npj91mvo5OIEB53xUNUXsKtzO16orqGBT-ysUGD29gR8Ef4q07_ee5hbfWU1t27O1u8_myYbEhSINR-N5qN-KcAHyxOSeVJ3oRwBZW0PF2fG68TtX1pVw0pNGTIAo_r2wh5hqj1nLvK4XlQLBO4PUciyXYSQTixEsOqDETj5zw1sPHKKfYeyNAH5Binj6uTItz7RqHYqExmeLuJ1EWNx7uYdTNywi4YYSqqMJyRvRqarW6sK0WIEZJqZW_fx5LRQjfAhKtAofMmEnh1cmK9EmIP8pKymFv6Cr1Yk1nHfb0FlrXEDxKSvkE1mNDELmn2sKiTzJZ7NRaZwVlqAlKSXx5WmNODn3lFElw-zyk8kMhmBm_CrmDFDoY4oUpY281gnRIIESdnGekeehKLqdP5tHqN8M6h7AONyKVJWXSteU5IogwswTeSgzK9mctgTE9EfCIVJ2ejRQ3BpxVxEhoWwvPyxekJ0IgQXxF1GENjZnZ31XnkVfjiUyrR5w8NE4gvlXW9NzWKaJIoqOsmxa7Zg8e5D6Crkji-ahrURY9jE9jod6Lfi_eGcv94O6cNy8sdqlFH9TCA2xUAuRZmZID3vFhKycjCmXxD3Kedk3zYq7u9ufExQ8lwqwj0-8yIOL5yCGcNCOdrkk998LAOn8m9xwuRJmj5Gz6WOqmKkLmVqyrrbT2Q1c0waI6BTCBeL1gvT_n0pz0gUX3om9e_qx4W_XDGQ39H1QebyFVcOJcbLTZWDIthUMVo9IdrJhuR0hd8cIX9Iyr5qTmazUoGnBnoyyaB-NMg0dWvVh9RMTxN_hhWScPRpnoTZWqfoxUE0k1NTLzJmBiTNx-JF-eR_WNF8tkCSRG7RHIOmYTXiVp23xf6_jP8L0MJfVPcABTMAAAAASUVORK5CYII="
	cMedia_Logo72Png = "0SNMiVBORw0KGgoAAAANSUhEUgAAAEgBBPD9CAYAAABV7bNHAAAAAXNSR0IArs4c6QAAAAlwSFlzAAALEwAACxMBAJqcGAAABCJpVFh0WE1MOmNvbS5hZG9iZS54bXAAAAAAADx4OnhtcG1ldGEgeG1sbnM6eD0iYWRvYmU6bnM6bWV0YS8iIHg6eG1wdGs9IlhNUCBDb3JlIDUuNC4wIj4KICAgPHJkZjpSREYgeG1sbnM6cmRmPSJodHRwOi8vd3d3LnczLm9yZy8xOTk5LzAyLzIyLXJkZi1zeW50YXgtbnMjIj4KICAgICAgPHJkZjpEZXNjcmlwdGlvbiByZGY6YWJvdXQ9IiIKICAgICAgICAgICAgeG0BqAh0aWYZaghucy4J1wxjb20vARoQLzEuMC9ONwAIZXhpWjcAARpiNwAUZGM9Imh0BdYMcHVybAXUOGRjL2VsZW1lbnRzLzEuMVJvAAh4bXAVOTKlAAh4YXAJbTEAIUsBukQ6UmVzb2x1dGlvblVuaXQ-MjwF0T4XAAAKATEFAQk2MENvbXByZXNzaW9uPjUNMy4UAD4wAABYGWcEPjcRZC4VAD4xABhPcmllbnRhAZgEPjENYS4UAD4wAABZTmEALhUAHTEheyw6UGl4ZWxYRGltZW4FxgGWUhkAPjkAJENvbG9yU3BhY2UBmQUzHRM-LgAFZwBZXmcAHRkdOSRkYzpzdWJqZWN0HRYhsRxyZGY6QmFnLx0XBDwvUi4AkDx4bXA6TW9kaWZ5RGF0ZT4yMDE2LTA2LTEwVDEwOjA2OjczPC86JAAdggEaLENyZWF0b3JUb29sPgXGAG0BDwwgMy41CTouIAANOwQ8LwGiAER5VgUZCRYYUkRGPgo8L5UA9GUNPgpGrmNuAAANSElEQVR4Ae1bCXhN1xZe9yYSElNIzENIiSmoqarU1Bra96oa-j5NNYby1FjVIr5-mqf6UFT18VFjVVFD9ZmKenhaDUWpmmsK0piJEESG89Z_2Dd7n3vuzT2R5EVltbGntaf_rr332mvtQ5RP-QjkI5CDCNjcth05ojbZ7O1Js4WSTfN1y5ubhZq2kRaMX5IbXXq77KTHqE9J0waRRnbif_B_3iH7NR5LrgDEkzehHlFvMzhDuMS83KRKi-rB1K_1U_T31k3o2dAqJhw5nPVGVGRO9GAuQRq9bqWzkFIlaMuIN8nb6z6emqZRyPBJdPrydSvNWODVnLcGO02lN6KeoYJXB9CsWSkWGnPL6kJCbCFuaxkKGwaXd4CDIpvNRrXLlzZw5ULSRn0oueRm6hkVlF29mQNkM_mFuMcCXl70bscWNKBtU7IzCIKCAwNE1BFWLlncEUfkxXqhtGxAN6pVrpSSnwOJFnyo7KYeo-pnR9vmAJm0XKqoP_0wqg9N_FtHmtb9JVo1pDsVKehLXnY7NahczqlG05BKOoiFfAro_GuHRlLXxmG0deSbOS9dmlaZD5WfeMl1cRqYxYwMMZAr9ohK4A6KyVnfv9eLnq_9hJxFh-MvkQ9L1ROlSyr5InH0_GXe6zWqaZCaExevUrURkwVbVsJPacG4oUpFkzFzOc7eD5k3-kGcA2tkvkmbtJGSluaUm9lyqVHWfCvYdzbeqa0cyoAAjKbIUWGUfq87LZyUZLUfj5fYsCXf0b1UZ5CsdhifkEhvLVhltdpD8mudyV4ghnqMDLbakMcAYbl8tGar1fYVfiy3yNkr6Oqt20p-LiXqkmbfzSdcSyv9eQwQGp20_kdKSr5n2v6ZKwn0VcyvtHjHfoq7dsOUJ_FuMu08cda0LHcytUBKp00sSf087S_TPQg6TZlihakKH-WvPV2f_H19nNr-bFMMvbd0vWMJFizgTVMj_kJ9WzVReIsVKkgTXu1AC2P2USwDejHxlr6JK0w5nyjAasAMioyqS75Xh2SmVLoF6K_1a9D8N7tQycJ-Loe9_fdYenvxOmWid1NSqf-Xq_n4L0-NqpRX6vZnHQp_oBt37lL3z5fRml-PKjy5lHiL7pWsRd2iu9CS6Cuu-nS7xHq2aOgWHDS6eOd-BRzRUVp6On39836RNA0hUV1YN_q_kUYtySeZlcoRLgfhVoI8GfiFGzddsl24cctlmSiQFHKRlbUwPb0ua6Zuf3CXDXt73XFV5hagxDvJruo58htXqUDf_nLYkZYjKMuMrt1yObbMqqrlX07Ikd3fLUDDl62nI3y8QyF8onQJ_YqAZSHToOea0Zxte-jUZZhoMghKIkwfRrqedIfbvESxlxPot7gL9PnWn40seSrt8VUDo25fpxpteLen0wQu30yi4XyKbTp0Qr9_dQirTuP5tCrhX0jhTU5NpQajp-lXFKXAemIReaVPdFRL9cKVIutkT8uoH1zoEEVHp4vG3EqQYBLhzpPn6Pa9FPLjC6hMQUX89dNOzjOL24j_y55NJ4LS7BGOPmwZ83PkWYlo0tYVSwFcNUFUl0pElutwRmQnJ3BcczuX-Hh7MZDhugXAuTRv5ngMUGTzBtStab2HngU27mEdmj90O7nVgMcADX6-WbaNKYI18keFPAbIeIeCFvw6a8F95q-k8wnOuhDyopZvpBc--ULXmAUg2KhHLt8gknk-9HiTHvTVGipSyJe6N3uScL0AOGeu3t_LQssE6aZYebZzf9hD49dt07PajJ-jWyADeTMP_9ciWv_b7zJrno57DFA6myremLWcxq_dRjB9IC3o7AOgRBqhrBftPRNPVd-bRL68Sd9yYQ2Q6-aluMcAiUHDzGqk01ec3TunLqmKIyySZlZJY1t5LW0ZILMJ_HDsNB2Mu0jlA4rqes7-s-dpx8mzZqyPXF62AIQ7W9j7Ux-5yXsyYI9PMU8a-zPyWAYIfrCXnqxJTapWUJyHeREcX2_vTO1ZmY3b8hKb3bMzwZAGWrnnEIVPW-S2D9z--7ZqTIXZybjl8EnaxvuVTJXYAzu31ysE93Xc9Rs0ffNOvuHv0lkKs3m3OT-KKM0mX_R1k23aZYsXoafZKYkfqDH_YXnjjgiz7x2-Jwoa88pzNLRdc-7Xh47xqdtr7jcUkwV7uGWAZJv0taTMvRMA52P2xoJGd2pDU7-PYRPtWjEP-oDznnvgkAzg2__MyJfp-IWrdPziFdr1QX-2hxfReYe2e4ausDekTc2qThfelxvUooin61G7ifNJGPCaVwvWwUHlUDa9LO3fjSq-M8HRr6cRy0tMbnj78TNy0jRev5LqlsbylOmUyQsQSAns4AIc8NerVJba1gpxAke0FVahDLu264gkzftxjyOOyPoDWVNOLQPkZTc3ISmjkRIwjsm0-fAJOalPJDXNYX7Ry36J_YN-ZVUh1kS_AgOW0kw2tMH0IpPsb7uUqDpRU7Lo9LQMkDdv0lbIyL_rVJxSHXc2cWURBals8AcdO-_sbICG_tSYGbp3NpHvgzL9yFcgQbBcyoTlmxWyvAfhFLNCeN0hk2bylk9-SgNeeERAiXdVAA6wiVbsM3jRJi_BmBNn6JzksDTuj8X9cgkg8YpMn4HhH4AnJieKjEvSz8fZ8Wj0u4kld-N2BkA4wVqNm03XHkjG2PDnRRd6OHLZRiUt-ERmgL9qSxf5mYXZIkF1K5ahcV3bU6saVQkX13e__o7W7T-m951i2F_wpkgmWBmLspVAprT0-xdh2aty6-49Bzgw3MnvINGXvLzQFsDFhVpIZ0AWJcjaeuGOjRLRmo9dHMcv1A3VzbHwZsAbW9zv_i920-A6KuangmGUHkxOSCF8-YKE9QDvIaEKCIIXFw4DI4FflsDiWdyDrAMk-eZK-vvRNwMj2IyhCiKM-MM6tNDHbDRvBBb2V-ZiTKNQ1JEnCNAgbdBnZImDtJpZGNCOvMxyTYLs0jE_rGNzEqcDNsi-87_FuHSC6weEvUMmGM1kCiyi-v3BL16QyKcUlh0ePkDjFrR63xFd8xZpYyifZADX6I0x8pulrUuQBFC54kX1NvHrduErx-xtu0kMqkFwOX2ZYe-QKcgAiHGJxV9PdLDLSwymlLdZmxYETRvXB3eUHSeZZYDEpicPbBG_CRL6jVD1wVetdKBij0Yd-WhG2rjE4iX7trzEIAGCoEC2nTA304dY8hJDXSHtoh1PQssAwfknE6Rn7Ootjqyk5AztFp5VmGdlqhBQTPGL4e2RTHiiJ0g-xUQeNuV2E-cpOo8oM4ZCmkV-rgBkVPTw_OU4v1oVJJtVsXxw7CdI-gz0qHJ81xJUq3wpEdVDuS2joggGPM6a1zucqgTBAeqenJeYdV3IsgSJ41YMDUZ8mWRNW1wZoAHLVDWohCOJS6ZM8JgIMpMglMEEsv_DwQ6zi-A3hsaXI1k5yawD9ECJw2BwLzIesYV8Mo58vFsE_XZOBSiMFUsQ1IHQMoF6HP9A-nZINht5D0I5HoEKgsIJSVo5KIKMJ6HgMUoQlhj2steb1de_LVkx8DWHvibqGEPLAAmpQEObDqo3c-SVYN1IkLiEGrXcpiEVdZb2YdUU88Xe2Hjlhp7Et3VZYvEOcuuRU6J5PezcsDYdGDtEV1SVAk5cv61eWGHoi5sykhb2fZWg4IY3qkPTu3cyVlPSlgFKTklzNHDyUsbeg0xcTMX-Ag8qHmmC1rC-Ih_3HVnrxqcN77RXffT_3qs-xILEyHoUNui2H8-lgQtXO3QltI-Tcd07kbqGDakUJI8VefXZpiSXI69rkzrKoYE8mSwDdCcl45SSN180iuuGeN7yB-szYknAbrNizwFHvzjd_pgSRU9K33iAZ9Z_75taBSM-iJG19DoVSuttwixb9_3PaNtR1XyLB1tnJg_XlxDa6P1sI9GUy_AnNvqJq40Zk2WArkmPwI0nWqcGGdZCnDYyffDtZkUajFaBCeymNuoteEYstyM_KsX-13rCHBqyaK1ii4YUz2EbN75MqldJPQDEeCCVUD9W7D5Ir81cKrJNQ8sA4X2zoJahVUVUD6f_ZyfBGrjlyEnqPXelUobjfjD7981-rQ1sDh27eqvCj8QnG7crLmxcN2RrACQUxvo-0hUH7U9ev13f8DEGeQ87wl7h2vyladF-_6CaUVOo6_TFpg8v5IGoWp8oMf9yRi_F3rFr9AAK4hAKG0TUCsHQFd25rf7eEdbEJaxHTd6w3RQ4tIvl-AJ_a4b9Y9XeI05mDdF3m5ohuqaMH0dWEOtVLEvhjWvrP9yafUcVwERdJbT5BtAX0Q4psAwQGsO-ANHHE5g_HRkAUjcKD2eLEwp_jwNZ3oMeB1DkOeYDJKNhEs8HyAQUOSsfIBkNk3g-QCagyFn5AMlomMTzATIBRc4yB0ijOJnp8YnbEinpkPLo2wVANvUi9bggpGmraPnyDHsOz9scoLM-Y9husepxweXBPHeQd9oA45zN72KCq8fIDvyF8IucrM7ODNVnLHge5VCjFJ7XSZ7jFlrwz-U8lQyb7qM8r_yx5yEE_gdNAtHUtaN5ygAAAABJRU5ErkJggg=="
)

var staticFiles = map[string]string{
	"/media/logo36.png": cMedia_Logo36Png,
	"/media/logo72.png": cMedia_Logo72Png,
}

// Lookup returns the bytes associated with the given path, or nil if the path was not found.
func Lookup(path string) []byte {
	s, ok := staticFiles[path]
	if ok {
		d, err := base64.URLEncoding.DecodeString(s)
		if err != nil {
			log.Print("main.Lookup: ", err)
			return nil
		}
		r, err := snappy.Decode(nil, d)
		if err != nil {
			log.Print("main.Lookup: ", err)
			return nil
		}
		return r
	}
	return nil
}

// ServeHTTP serves the stored file data over HTTP.
func ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p := r.URL.Path
	if strings.HasSuffix(p, "/") {
		p += "index.html"
	}
	b := Lookup(p)
	if b != nil {
		mt := mime.TypeByExtension(path.Ext(p))
		if mt != "" {
			w.Header().Set("Content-Type", mt)
		}
		w.Header().Set("Expires", time.Now().AddDate(0, 0, 1).Format(time.RFC1123))
		w.Write(b)
	} else {
		http.NotFound(w, r)
	}
}
