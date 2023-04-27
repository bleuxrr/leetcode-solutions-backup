package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
)

var lang = "golang"
var cookie = ""
var client *http.Client

var ext = map[string]string{
	"golang": ".go",
}

type question struct {
	questionId         int //qid
	questionFrontendId int
	// title              string
	titleSlug string
}

func getCookie() string {
	configFile, err := os.Open("config.json")
	if err != nil {
		log.Fatal(err)
	}
	defer configFile.Close()
	byteValue, _ := io.ReadAll(configFile)
	var config map[string]interface{}
	json.Unmarshal([]byte(byteValue), &config)
	return config["cookie"].(string)
}

func queryQuestionList(skip, limit int, client *http.Client) []byte {
	url := "https://leetcode.com/graphql/"
	query := `{"query":"\n    query problemsetQuestionList($categorySlug: String, $limit: Int, $skip: Int, $filters: QuestionListFilterInput) {\n  problemsetQuestionList: questionList(\n    categorySlug: $categorySlug\n    limit: $limit\n    skip: $skip\n    filters: $filters\n  ) {\n    total: totalNum\n    questions: data {\n      acRate\n      difficulty\n      freqBar\n      frontendQuestionId: questionFrontendId\n      isFavor\n      paidOnly: isPaidOnly\n      status\n      title\n      titleSlug\n      topicTags {\n        name\n        id\n        slug\n      }\n      hasSolution\n      hasVideoSolution\n    }\n  }\n}\n    ","variables":{"categorySlug":"","skip":` + strconv.Itoa(skip) + `,"limit":` + strconv.Itoa(limit) + `,"filters":{"status":"AC"}},"operationName":"problemsetQuestionList"}`
	req, err := http.NewRequest("POST", url, bytes.NewReader([]byte(query)))
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Add("cookie", cookie)
	req.Header.Add("content-type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	return body
}

func getQuestionListTotal(qList *[]byte) int {
	var jsonMap map[string]interface{}
	json.Unmarshal(*qList, &jsonMap)
	d := jsonMap["data"]
	dataMap := d.(map[string]interface{})
	p := dataMap["problemsetQuestionList"]
	problemsetQuestionList := p.(map[string]interface{})
	return int(problemsetQuestionList["total"].(float64))
}

func getQuestionList(qList *[]byte) []question {
	var jsonMap map[string]interface{}
	json.Unmarshal(*qList, &jsonMap)
	d := jsonMap["data"]
	dataMap := d.(map[string]interface{})
	p := dataMap["problemsetQuestionList"]
	problemsetQuestionList := p.(map[string]interface{})
	questions := problemsetQuestionList["questions"]
	questionsArray := questions.([]interface{})

	ret := make([]question, len(questionsArray))
	for i := 0; i < len(questionsArray); i++ {
		problem := questionsArray[i].(map[string]interface{})
		ret[i].titleSlug = problem["titleSlug"].(string)
	}
	return ret
}

func queryQuestionInfo(titleSlug string, client *http.Client) []byte {
	url := "https://leetcode.com/graphql/"
	query := `{"query":"\n    query questionTitle($titleSlug: String!) {\n  question(titleSlug: $titleSlug) {\n    questionId\n    questionFrontendId\n    title\n    titleSlug\n    isPaidOnly\n    difficulty\n    likes\n    dislikes\n  }\n}\n    ","variables":{"titleSlug":"` + titleSlug + `"},"operationName":"questionTitle"}`
	req, err := http.NewRequest("POST", url, bytes.NewReader([]byte(query)))
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Add("cookie", cookie)
	req.Header.Add("content-type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	return body
}

func updateQuestion(qu *question, body *[]byte) {
	var jsonMap map[string]interface{}
	json.Unmarshal(*body, &jsonMap)
	d := jsonMap["data"]
	dataMap := d.(map[string]interface{})
	q := dataMap["question"]
	questionInfo := q.(map[string]interface{})
	qid, _ := strconv.Atoi(questionInfo["questionId"].(string))
	qfid, _ := strconv.Atoi(questionInfo["questionFrontendId"].(string))
	qu.questionId = qid
	qu.questionFrontendId = qfid
	// qu.title = questionInfo["title"].(string)
}

func querySolution(qid int, client *http.Client) []byte {
	url := "https://leetcode.com/submissions/latest/?qid=" + strconv.Itoa(qid) + "&lang=" + lang
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Add("cookie", cookie)
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	return body
}

func saveSolution(body *[]byte, dirName string, fileName string) {
	var jsonMap map[string]interface{}
	json.Unmarshal(*body, &jsonMap)
	str := jsonMap["code"].(string)
	data := []byte(str)
	err := os.MkdirAll("solutions/"+dirName, 0750)
	if err != nil {
		log.Fatal(err)
	}
	err = os.WriteFile("solutions/"+dirName+"/"+fileName, data, 0660)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	cookie = getCookie()
	client = &http.Client{}

	skip := 0
	limit := 50
	qList := queryQuestionList(skip, limit, client)
	total := getQuestionListTotal(&qList)
	fmt.Printf("backup %d solutions\n", total)

	//get all ac questions
	questionList := getQuestionList(&qList)
	for i := 1; i < (total+49)/50; i++ {
		skip = i * 50
		qList = queryQuestionList(skip, limit, client)
		questionList = append(questionList, getQuestionList(&qList)...)
	}

	//get questionId and questionFrontendId
	for i := 0; i < len(questionList); i++ {
		body := queryQuestionInfo(questionList[i].titleSlug, client)
		updateQuestion(&questionList[i], &body)
		// fmt.Printf("query %d/%d problem info\n", i+1, total)

		body = querySolution(questionList[i].questionId, client)
		if len(body) == 0 {
			fmt.Printf("%d/%d problem %s doesn't support %s language\n", i+1, total, questionList[i].titleSlug, lang)
			continue
		}
		questionFrontendId := fmt.Sprintf("%04d", questionList[i].questionFrontendId)
		dirName := questionFrontendId + "-" + questionList[i].titleSlug
		// dirName = strings.ReplaceAll(dirName, " ", "-")
		fileName := questionFrontendId + "-" + questionList[i].titleSlug + ext[lang]
		saveSolution(&body, dirName, fileName)
		fmt.Printf("%d/%d problems saved\n", i+1, total)
	}

}
