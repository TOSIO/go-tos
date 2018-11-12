package httpSend

import (
	//"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
)

func SendHttp(urlString string, send string) ([]byte, error) {
	resp, err := http.Post(urlString, "application/json", strings.NewReader(send))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, err
}
