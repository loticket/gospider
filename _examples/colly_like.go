package main

import (
	"github.com/PuerkitoBio/goquery"
	"github.com/zhshch2002/goreq"
	"github.com/zhshch2002/gospider"
)

/* colly example http://go-colly.org/docs/examples/basic/
// Instantiate default collector
c := colly.NewCollector(
	// Visit only domains: hackerspaces.org, wiki.hackerspaces.org
	colly.AllowedDomains("hackerspaces.org", "wiki.hackerspaces.org"),
)

// On every a element which has href attribute call callback
c.OnHTML("a[href]", func(e *colly.HTMLElement) {
	link := e.Attr("href")
	// Print link
	fmt.Printf("Link found: %q -> %s\n", e.Text, link)
	// Visit link found on page
	// Only those links are visited which are in AllowedDomains
	c.Visit(e.Request.AbsoluteURL(link))
})

// Before making a request print "Visiting ..."
c.OnRequest(func(r *colly.Request) {
	fmt.Println("Visiting", r.URL.String())
})

// Start scraping on https://hackerspaces.org
c.Visit("https://hackerspaces.org/")
*/
func main() {
	s := gospider.NewSpider(goreq.WithFilterLimiter(false, &goreq.FilterLimiterOpinion{
		LimiterMatcher: goreq.LimiterMatcher{Glob: "*.hackerspaces.org"},
		Allow:          true,
	}, &goreq.FilterLimiterOpinion{
		LimiterMatcher: goreq.LimiterMatcher{Glob: "hackerspaces.org"},
		Allow:          true,
	}))

	// On every a element which has href attribute call callback
	s.OnHTML("a[href]", func(ctx *gospider.Context, sel *goquery.Selection) {
		link, _ := sel.Attr("href")
		// Print link
		ctx.Printf("Link found: %q -> %s\n", sel.Text(), link)
		// Visit link found on page
		// Only those links are visited which are in AllowedDomains
		ctx.AddTask(goreq.Get(link)) // gospider will auto convert to absolute URL
	})

	s.SeedTask(goreq.Get("https://hackerspaces.org/"))
	s.Wait()
}
