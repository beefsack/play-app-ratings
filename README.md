# play-app-ratings - Fetch Google Play app ratings

I built this tool because I have hundreds of Android games from Humble Bundle
and I'm not sure which ones I want to play first.  **play-app-ratings** accepts
app names on STDIN, one per line, and outputs a CSV with ratings and rating
counts to STDOUT.

For my own case, I then applied a weighted average by hand and discovered I
should play Kingdom Rush first, and it turned out to be a really well made
game!

## Installation

You must have [Go](https://golang.org/doc/install) installed and configured,
after which you can run:

```bash
go get github.com/beefsack/play-app-ratings
play-app-ratings < game_list > ratings.csv
```
