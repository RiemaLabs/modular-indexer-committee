package apis

import "encoding/base64"

func BatchDecodeBase64(strs []string) ([][]byte, error) {
	res := make([][]byte, 0)
	for _, s := range strs {
		bytes, err := base64.StdEncoding.DecodeString(s)
		if err != nil {
			return res, err
		}
		res = append(res, bytes)
	}
	return res, nil
}
