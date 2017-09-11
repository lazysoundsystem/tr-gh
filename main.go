package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Machiel/slugify"
	"github.com/google/go-github/github"
	"github.com/orakili/go-trello-client"
	"golang.org/x/oauth2"
	"io/ioutil"
	"log"
	"net/url"
	"strings"
)

type Config struct {
	TrelloApiKey    string         `json:"TrelloApiKey"`
	TrelloUserToken string         `json:"TrelloUserToken"`
	TrelloBoardId   string         `json:"TrelloBoardId"`
	TrelloClient    *trello.Client `json:"-"`
	GithubToken     string         `json:"GithubToken"`
	GithubUser      string         `json:"GithubUser"`
	GithubRepo      string         `json:"GithubRepo"`
	GithubBranch    string         `json:"GithubBranch"`
	GithubClient    *github.Client `json:"-"`
	ItemPath        string         `json:"ItemPath"`
}

var config Config

var latestCommit *github.Commit
var newTree github.Tree
var treeEntry github.TreeEntry
var tree github.Tree
var gitService github.GitService
var newRef github.Reference
var ctx context.Context

// init read the configuration file and initialize trello and github clients
func init() {
	ctx = context.Background()
	data, err := ioutil.ReadFile("./config/config.json")
	if err != nil {
		log.Fatal("Cannot read configuration file.")
	}
	err = json.Unmarshal(data, &config)
	if err != nil {
		log.Fatal("Invalid configuration file.")
	}

	// Set up github client
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: config.GithubToken},
	)
	tc := oauth2.NewClient(oauth2.NoContext, ts)
	config.GithubClient = github.NewClient(tc)
	oldRef, _, err := config.GithubClient.Git.GetRef(ctx, config.GithubUser, config.GithubRepo, fmt.Sprintf("heads/%s", config.GithubBranch))
	if err != nil {
		fmt.Printf("error getting ref: %s", err)
		return
	}
	refSHA := oldRef.Object.SHA
	latestCommit, _, err = config.GithubClient.Git.GetCommit(ctx, config.GithubUser, config.GithubRepo, *refSHA)
	if err != nil {
		fmt.Printf("error getting repoCommit: %s", err)
	}

	// Set up trello client
	config.TrelloClient = trello.NewClient(config.TrelloApiKey, config.TrelloUserToken)
}

func main() {
	params := url.Values{}
	params.Add("key", config.TrelloApiKey)
	params.Add("token", config.TrelloUserToken)
	params.Add("lists", "open")
	params.Add("cards", "open")
	params.Add("card_fields", "name,desc,pos,idList,labels,due")
	params.Add("fields", "name,pos,desc")
	params.Add("card_attachments", "true")
	params.Add("card_attachment_fields", "name,url")

	board, err := config.TrelloClient.GetBoard(config.TrelloBoardId, params)
	if err != nil {
		log.Printf("Unable to retrieve Trello board: %s", err)
		return
	}

  // Set up variables
	treeMode := "100644"
	treeType := "blob"

	// Remove Private cards - convention
	// Add list name alongside list.Id
	listMap := make(map[string]string)
	for _, list := range board.Lists {
		if !strings.HasPrefix(list.Name, "PRIVATE") {
			listMap[list.Id] = list.Name
		}
	}
	lists, err := json.MarshalIndent(listMap, "", "\t")
	if err != nil {
		fmt.Printf("error marshaling listmap: %s", err)
		return
	}
	listsPath := "_data/lists.json"
	treeEntry.Path = &listsPath
	treeEntry.Mode = &treeMode
	treeEntry.Type = &treeType
	listsContent := string(lists)
	treeEntry.Content = &listsContent
	tree.Entries = append(tree.Entries, treeEntry)
	var filteredCards []trello.Card
	for _, card := range board.Cards {
		_, ok := listMap[card.IdList]
		if ok {

			filteredCards = append(filteredCards, card)
			// Write files for individual cards.
			cardText := fmt.Sprintf("---\nname: %q\nlayout: card\n---", card.Name)
			cardSlug := slugify.Slugify(card.Name)
			cardPath := fmt.Sprintf("_%s/%s/index.html", config.ItemPath, cardSlug)
			treeEntry.Path = &cardPath
			treeEntry.Mode = &treeMode
			treeEntry.Type = &treeType
			treeEntry.Content = &cardText
			tree.Entries = append(tree.Entries, treeEntry)
		}
	}
	cardsJson, err := json.MarshalIndent(filteredCards, "", "\t")
	cardsPath := "_data/all.json"
	treeEntry.Path = &cardsPath
	treeEntry.Mode = &treeMode
	treeEntry.Type = &treeType
	cardsContent := string(cardsJson)
	treeEntry.Content = &cardsContent
	tree.Entries = append(tree.Entries, treeEntry)

	newTree, _, err := config.GithubClient.Git.CreateTree(ctx, config.GithubUser, config.GithubRepo, *latestCommit.Tree.SHA, tree.Entries)
	if err != nil {
		fmt.Printf("error creating tree 200: %s", err)
		return
	}

	var commit github.Commit
	commit.Tree = newTree
	commitMessage := "updating cards"
	commit.Message = &commitMessage
	commit.Parents = append(commit.Parents, *latestCommit)
	newCommit, _, err := config.GithubClient.Git.CreateCommit(ctx, config.GithubUser, config.GithubRepo, &commit)
	if err != nil {
		fmt.Printf("error creating commit: %s", err)
		return
	}

	var object github.GitObject
	object.SHA = newCommit.SHA
	object.URL = newCommit.URL
	objectType := "commit"
	object.Type = &objectType
	var ref github.Reference
	ref.Object = &object
	refRef := fmt.Sprintf("refs/heads/%s", config.GithubBranch)
	ref.Ref = &refRef
	refURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/git/%s", config.GithubUser, config.GithubRepo, refRef)
	ref.URL = &refURL
	_, _, err = config.GithubClient.Git.UpdateRef(ctx, config.GithubUser, config.GithubRepo, &ref, false)
	if err != nil {
		fmt.Printf("error updating ref: %s", err)
		return
	}
	return
}

//  fmt.Printf("Card full: %#v \n", card)
//  fmt.Printf("Card name: %s on list: %s\n", card.Name, name)
//  //fmt.Printf("Card with hash: %#v \n", card)

//var inspect bytes.Buffer
//err = json.Indent(&inspect, body, "", "\t")
//if err != nil {
//	log.Printf("error indenting board: %s", err)
//	return
//}
//fmt.Printf("lists indented %s\n", string(inspect.Bytes()))

//dirPath := fmt.Sprintf("%s/%s", basePath, cardSlug)
//if _, err := os.Stat(dirPath); os.IsNotExist(err) {
//	os.Mkdir(dirPath, 0755)
//}
// write to /home/andy/ReliefWeb/podcast-site/_<item path>/<name>/index.html
//filePath := fmt.Sprintf("%s/index.html", dirPath)
//err = ioutil.WriteFile(filePath, []byte(cardText), 0644)
//if err != nil {
//	fmt.Printf("error writing file: %s", err)
//	return
//}
//if err != nil {
//	fmt.Printf("error marshaling cards: %s", err)
//	return
//}
//filePath := "/home/andy/ReliefWeb/podcast-site/_data/all.json"
//err = ioutil.WriteFile(filePath, cardsJson, 0644)
//if err != nil {
//	fmt.Printf("error writing file: %s", err)
//	return
//}
//oldRef, resp, err := gitService.GetRef(ctx, config.GithubUser, config.GithubRepo, fmt.Sprintf("heads/%s", config.GithubBranch))
//if err != nil {
//	fmt.Printf("error creating tree: %s", err)
//	return
//}
//fmt.Printf("%s\n", oldRef)
//fmt.Printf("%v\n", resp)

//err = ioutil.WriteFile(filePath, lists, 0644)
//if err != nil {
//	fmt.Printf("error writing file: %s", err)
//	return
//}
//basePath := fmt.Sprintf("/home/andy/ReliefWeb/podcast-site/_%s", config.ItemPath)
//if _, err := os.Stat(basePath); os.IsNotExist(err) {
//	os.Mkdir(basePath, 0755)
//}

//tree.Entries:
//[
//  github.TreeEntry
//  {SHA:"dda551fc2e915794e0f4fc9e1d3c19ee731cc159",
//   Path:"_data/all.json",
//   Mode:"100644",
//   Type:"blob",
//   Content:"{
//    "589b2a5f19b871bbb23bca64": "research-and-think-tanks",
//    "58d796b8474f66719c5156fa": "development",
//    "58d796c6d5cd68d0a879de28": "humanitarian-crises",
//    "58d796fc0fb9e7980313a187": "humanitarian-learning"
//  }
//  null"}
//  github.TreeEntry{SHA:"dda551fc2e915794e0f4fc9e1d3c19ee731cc159", Path:"_data/all.json", Mode:"100644", Type:"blob", Content:"{
//	"589b2a5f19b871bbb23bca64": "research-and-think-tanks",
//	"58d796b8474f66719c5156fa": "development",
//	"58d796c6d5cd68d0a879de28": "humanitarian-crises",
//	"58d796fc0fb9e7980313a187": "humanitarian-learning"
//}null"}]
