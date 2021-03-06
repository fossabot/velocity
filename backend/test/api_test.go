package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/DATA-DOG/godog/gherkin"
	"github.com/docker/go/canonical/json"
)

func sendPOSTWithAttrsTable(reqAttrs *gherkin.DataTable, uri string) error {
	payloadMap := map[string]string{}

	for _, r := range reqAttrs.Rows[1:] {
		k := r.Cells[0].Value
		v := r.Cells[1].Value
		payloadMap[k] = v
	}

	payloadJSON, _ := json.Marshal(payloadMap)

	req, _ := http.NewRequest(
		"POST",
		fmt.Sprintf("%s%s", testServer.URL, uri),
		bytes.NewBuffer(payloadJSON),
	)

	req.Header = headers

	response, _ = client.Do(req)
	responseBody, _ = ioutil.ReadAll(response.Body)
	response.Body.Close()

	return nil
}

func theResponseHasStatus(expectedStatus string) error {
	if response.Status == expectedStatus {
		return nil
	}

	return fmt.Errorf("expected: %s, got: %s", expectedStatus, response.Status)
}

func theResponseHasTheFollowingAttributes(expectedAttrs *gherkin.DataTable) error {

	resp := map[string]interface{}{}
	err := json.Unmarshal(responseBody, &resp)
	if err != nil {
		return err
	}

	for _, r := range expectedAttrs.Rows[1:] {
		eAttr := r.Cells[0].Value
		eType := r.Cells[1].Value
		eVal := r.Cells[2].Value

		val, err := recurseChars(eAttr, resp)
		if err != nil {
			return err
		}

		err = compareAttrs(val, eAttr, eType, eVal)
		if err != nil {
			return err
		}
	}

	return nil
}

func recurseChars(eAttr string, resp map[string]interface{}) (interface{}, error) {
	// var key string
	var keyAt int
	var index int
	var indexAt int
	var indexLength int

	// check for object
	re := regexp.MustCompile(`\.\w+`)
	loc := re.FindStringIndex(eAttr)
	if loc != nil {
		// key = eAttr[loc[0]+1 : loc[1]]
		keyAt = loc[0] + 1
	}

	re = regexp.MustCompile(`\[\d+\]`)
	loc = re.FindStringIndex(eAttr)
	if loc != nil {
		index, _ = strconv.Atoi(eAttr[loc[0]+1 : loc[1]-1])
		indexAt = loc[0] + 1
		indexLength = loc[1] - loc[0]
	}

	if (keyAt > 0 && indexAt > 0) && // both key and index found
		(keyAt < indexAt) { // key before index
		return recurseChars(eAttr[keyAt:], resp[eAttr].(map[string]interface{}))
	}

	if ((keyAt > 0 && indexAt > 0) && (keyAt > indexAt)) || // both key and index found and key after index
		(keyAt == 0 && indexAt > 0) { // only index found
		parts := strings.Split(eAttr, "[")
		// re := regexp.MustCompile(`[\d+]`)
		// in := re.FindString(eAttr)
		in := parts[0]
		switch x := resp[in].(type) {
		case []interface{}:
			if (indexAt + indexLength) < len(eAttr) {
				return recurseChars(eAttr[indexAt+indexLength:], x[index].(map[string]interface{}))
			}
			l := map[string]interface{}{}
			for i, v := range x {
				l[strconv.Itoa(i)] = v
			}
			return recurseChars(strconv.Itoa(index), l)
		default:
			return nil, fmt.Errorf("could not determine attribute: %s (%s). got %v. full: %+v", in, eAttr, x, resp)
		}
	}

	return resp[eAttr], nil

}

func compareAttrs(val interface{}, eAttr, eType, eVal string) error {
	var dVal interface{}
	var compareTypeFunc func(string, string, interface{}) error
	switch eType {
	case "string":
		dVal = ""
		compareTypeFunc = compareStringType
		break
	case "integer":
		dVal = 0
		compareTypeFunc = compareIntegerType
		break
	case "timestamp":
		dVal = time.Time{}
		compareTypeFunc = compareTimestampType
		break
	default:
		return fmt.Errorf("invalid type %s for %s", eType, eAttr)
	}

	dJSONAttr, _ := json.Marshal(val)
	err := json.Unmarshal(dJSONAttr, &dVal)
	if err != nil {
		return err
	}

	return compareTypeFunc(eAttr, eVal, dVal)
}

func compareStringType(eAttr, eVal string, dVal interface{}) error {
	v := dVal.(string)
	switch eVal {
	case "*any":
		if len(v) < 1 {
			return fmt.Errorf("%s was empty", eAttr)
		}
	default:
		if eVal != v {
			return fmt.Errorf("expected %s, got %s", eAttr, eVal)
		}
	}

	return nil
}

func compareIntegerType(eAttr, eVal string, dVal interface{}) error {
	v := dVal.(float64)
	eV, err := strconv.Atoi(eVal)
	if err != nil {
		return err
	}
	switch eVal {
	default:
		if eV != int(v) {
			return fmt.Errorf("expected %s, got %s", eAttr, eVal)
		}
	}

	return nil
}

func compareTimestampType(eAttr, eVal string, dVal interface{}) error {
	v, err := time.Parse(time.RFC3339, dVal.(string))

	if err != nil {
		return err
	}

	switch eVal[:1] {
	case "*":
		duration, err := time.ParseDuration(eVal[1:])
		if err != nil {
			return err
		}
		e := time.Now().Add(duration)
		if v.Format(time.ANSIC) != e.Format(time.ANSIC) {
			return fmt.Errorf("expected %s, got %s", e, v)
		}
	default:
		if eVal != dVal.(string) {
			return fmt.Errorf("expected %s, got %s", eVal, dVal)
		}
	}

	return nil
}
