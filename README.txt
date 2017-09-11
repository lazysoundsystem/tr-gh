A simple framework to take cards from a trello board, parse them to yaml and
json files, and commit them to gh-pages repo.

The use case is for trello as a content management system, with github-pages as
the front end.

Uses golang: orakili/go-trello-client and google/go-github.

Config:
  TrelloApiKey
  TrelloUserToken
  TrelloBoardId

  GithubAccessToken
  GithubUser
  GithubRepo

  ItemPath - path to be used on the frontend.

Requires all frontend work to be in the same repository, using jekyll and liquid
templates.

Roadmap:
  Board set-up
    list title as category
    attached images as images
    attached links as links - with title
    description as body
    labels as tags
  Lists with 'PRIVATE' at the start of their name are not processed.

  MVP:
    get all data from board
      create a json file mapping list names to their ids
      create a json file of all the cards
      create a yaml file for each card
    commit to github repo

  Template
