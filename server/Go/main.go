package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/labstack/echo/v4"
)

func main() {
	token, clientId := getTwitchTokenAndClientID()
	users, err := getUsersFromJson()
	if err != nil {
		return
	}

	e := echo.New()
	//CORS設定でlocalhost:3000とlocalhost:3001を許可
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Response().Header().Set("Access-Control-Allow-Origin", "http://localhost:3001")
			c.Response().Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Response().Header().Set("Access-Control-Allow-Headers", "Content-Type, Access-Control-Allow-Headers, Authorization, X-Requested-With")
			c.Response().Header().Set("Access-Control-Allow-Credentials", "true")
			return next(c)
		}
	})

	e.Static("/", "static")

	e.GET("/api/get_streaming_user", func(c echo.Context) error {
		fmt.Println("get_streaming_user")
		userDataList := []map[string]interface{}{}

		for _, user := range users {
			userId, err := getTwitchUserID(user, clientId, token)
			if err != nil {
				fmt.Println(err)
				return err
			}
			fmt.Println("a")
			urlStr := "https://api.twitch.tv/helix/streams?user_id=" + userId
			req, _ := http.NewRequest("GET", urlStr, nil)
			req.Header.Set("Client-Id", clientId)
			req.Header.Set("Authorization", "Bearer "+token)

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				fmt.Println(err)
			}
			defer resp.Body.Close()
			byteArray, _ := io.ReadAll(resp.Body)

			type TwitchResponse struct {
				Data []struct {
					Type        string `json:"type"`
					ID          string `json:"id"`
					UserID      string `json:"user_id"`
					UserLogin   string `json:"user_login"`
					UserName    string `json:"user_name"`
					Title       string `json:"title"`
					ViewerCount int    `json:"viewer_count"`
				} `json:"data"`
			}

			var TwitchResponseData TwitchResponse
			if err := json.Unmarshal(byteArray, &TwitchResponseData); err != nil {
				fmt.Println(err)
			}

			if len(TwitchResponseData.Data) > 0 {
				userData := map[string]interface{}{
					"userId":      TwitchResponseData.Data[0].UserID,
					"userLogin":   TwitchResponseData.Data[0].UserLogin,
					"userName":    TwitchResponseData.Data[0].UserName,
					"title":       TwitchResponseData.Data[0].Title,
					"viewerCount": fmt.Sprintf("%d", TwitchResponseData.Data[0].ViewerCount),
				}
				userDataList = append(userDataList, userData)
				fmt.Println("userDataList");
				fmt.Println(userDataList);
			}
		}
		fmt.Println(userDataList)

		return c.JSON(http.StatusOK, map[string][]map[string]interface{}{
			"users": userDataList,
		})
	})

	e.Start(":3000")

}

func getTwitchUserID(loginName string, clientId string, token string) (string, error) {
	urlStr := "https://api.twitch.tv/helix/users?login=" + loginName
	req, _ := http.NewRequest("GET", urlStr, nil)
	req.Header.Set("Client-Id", clientId)
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	defer resp.Body.Close()
	byteArray, _ := io.ReadAll(resp.Body)

	usersMap := map[string]interface{}{}

	if err := json.Unmarshal(byteArray, &usersMap); err != nil {
		fmt.Println(err)
		return "", err
	}
	fmt.Println(usersMap)

	if (usersMap["data"] != nil) && (len(usersMap["data"].([]interface{})) > 0) {
		userData := usersMap["data"].([]interface{})[0].(map[string]interface{})
		fmt.Println(userData)
		return userData["id"].(string), nil
	}
	return "", nil
}

// UrlにPOSTする
func getTwitchTokenAndClientID() (string, string) {
	clientId, clientSecret := getKeyFromJson()

	if clientId == "" || clientSecret == "" {
		return "", ""
	}

	urlStr := "https://id.twitch.tv/oauth2/token"
	urlStr += "?client_id=" + clientId
	urlStr += "&client_secret=" + clientSecret
	urlStr += "&grant_type=client_credentials"
	urlStr += "&scope=channel:read:subscriptions "
	resp, err := http.PostForm(urlStr, nil)
	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()
	byteArray, _ := io.ReadAll(resp.Body)
	fmt.Println(string(byteArray))

	var jsonMap map[string]string
	if err := json.Unmarshal(byteArray, &jsonMap); err != nil {
		fmt.Println(err)
	}
	fmt.Println(jsonMap["access_token"])
	return jsonMap["access_token"], clientId
}

// key.jsonからclient_idを取得する
func getKeyFromJson() (string, string) {
	file, err := os.Open("key.json")
	if err != nil {
		fmt.Println(err)
		return "", ""
	}
	defer file.Close()

	key := map[string]interface{}{}

	if err := json.NewDecoder(file).Decode(&key); err != nil {
		fmt.Println(err)
		return "", ""
	}

	client_id, id_ok := key["client_id"].(string)
	client_secret, secret_ok := key["client_secret"].(string)

	if !id_ok || !secret_ok {
		fmt.Println("key.jsonにclient_idとclient_secretがありません")
		return "", ""
	}
	fmt.Println(client_id)
	fmt.Println(client_secret)

	return client_id, client_secret
}

func getUsersFromJson() ([]string, error) {
	file, err := os.Open("users.json")
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	defer file.Close()

	users := map[string]interface{}{}

	if err := json.NewDecoder(file).Decode(&users); err != nil {
		fmt.Println(err)
		return nil, err
	}

	usersList, ok := users["users"].([]interface{})
	if !ok {
		fmt.Println(usersList)
		fmt.Println("users.jsonがおかしいです")
		return nil, err
	}

	usersListStr := []string{}
	for _, user := range usersList {
		userStr, ok := user.(string)
		if !ok {
			fmt.Println("users.jsonがおかしいです")
			return nil, err
		}
		usersListStr = append(usersListStr, userStr)
	}
	fmt.Println(usersListStr)

	return usersListStr, nil
}
