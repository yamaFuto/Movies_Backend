package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

type JSONResponse struct {
	Error   bool        `json:"error"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func (app *application) writeJSON(w http.ResponseWriter, status int, data interface{}, headers ...http.Header) error {
	out, err := json.Marshal(data)
	if err != nil {
		return err
	}

	if len(headers) > 0 {
		//引数でなくてもいいようにspliteを書いているから、[0]をつける必要がある
		for key, value := range headers[0] {
			w.Header()[key] = value
		}
	}

	w.Header().Set("Content-Type", "application/json")
	//二個目以降のheaderをheaderにapeendする
	w.WriteHeader(status)
	_, err = w.Write(out)
	if err != nil {
		return err
	}

	return nil
}

func (app *application) readJSON(w http.ResponseWriter, r *http.Request, data interface{}) error {
	maxBytes := 1024 * 1024 // one megabyte
	//Bodyをbyte[]型でサイズ制限をして返す(容量を超えたらResponseWriterによってconnectionが封鎖される)
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

	//r.Bodyをdecodeするmethodを生成
	//Decoderにio.Readerを渡してDecodeする方法は、
	//Decodeはストリームから次のjsonを取り出して処理するため、jsonが複数個含まれているファイルも処理できる
	//unmarshalと違ってerrorなどをカスタマイズできる、またioのFileを引数にとることができる
	//Unmarshalに渡すバイト列はひとつのjsonとして正しい形式である必要がある(jsonは一つだけ)
	dec := json.NewDecoder(r.Body)

	//いれるdataの構造と違うkeyを持っている場合にはerrorを返すようになるようにする
	dec.DisallowUnknownFields()

	err := dec.Decode(data)
	if err != nil {
		return err
	}

	//requestがjsonを2つ以上持っていた場合なerrorを返す
	err = dec.Decode(&struct{}{})
	//jsonが一つの時はio.EOFと同じerrorをDecode()が吐く
	if err != io.EOF {
		return errors.New("body must only contain a single JSON value")
	}

	return nil
}

func (app *application) errorJSON(w http.ResponseWriter, err error, status ...int) error {
	//もし第三引数が設定されていなかった時のためにdefaultとして設定
	statusCode := http.StatusBadRequest

	if len(status) > 0 {
		statusCode = status[0]
	}

	var payload JSONResponse
	payload.Error = true
	payload.Message = err.Error()
	
	return app.writeJSON(w, statusCode, payload)
}

