package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/gocolly/colly"
	"github.com/xuri/excelize/v2"
)

const (
	listLink    string = "https://www.work.ua/ru/jobs/by-company/by-industry/it/?page="
	companyLink string = "https://www.work.ua"
	xmlString   string = "/html/body/section/div/div[3]"
)

// Company структура с данными, которые программа собирает про компанию
type Company struct {
	name        string // название компании
	website     string // веб-сайт компании
	workUA      string // ссылка на work.ua компании
	description string // короткое описание компании
	placesCount int    // количество вакансий в Киеве
}

// IncrementPlaces ...
func (c Company) IncrementPlaces() Company {
	c.placesCount++
	return c
}

// NewCompany функция-генератор новой компании
func NewCompany(name, website, workUA, descr string) Company {
	log.Printf("creating new company: %v\n", name)
	return Company{
		name:        name,
		website:     website,
		workUA:      workUA,
		description: descr,
	}
}

func main() {

	// companies карта в которой будут хранитья компании
	// в роли ключа выступает ссылка на work.ua
	var companies map[string]Company = map[string]Company{}

	for page := 1; page <= 127; page++ {
		log.Printf("new loop: page %v\n", page)

		// pageCollector собирает с страницы-списка ссылки на компании
		pageCollector := colly.NewCollector()

		// companyCollector собирает со страницы компании всю информацию
		companyCollector := colly.NewCollector()

		// cityCollector собирает информацию про доступные вакансии и города
		cityCollector := colly.NewCollector()

		companyCollector.OnXML("/html/body/section/div/div/div[1]/div[2]", func(e *colly.XMLElement) {
			companies[e.Request.URL.String()] = NewCompany(
				e.ChildText("div/h1"),
				e.ChildAttr("div/div/div/p/span/a", "href"),
				e.Request.URL.String(),
				fmt.Sprintf("%v %v", e.ChildText("div/p[1]"), e.ChildText("div/p[2]")))
		})

		pageCollector.OnXML(fmt.Sprintf(xmlString), func(e *colly.XMLElement) {
			for companyCard := 2; companyCard <= 23; companyCard++ {
				link := companyLink + e.ChildAttr(fmt.Sprintf("div[%v]/div/div[2]/h2/a", companyCard), "href")
				companyCollector.Visit(link)
				cityCollector.Visit(link)
			}
		})

		cityCollector.OnHTML("div.card.card-hover.card-visited.wordwrap.job-link", func(e *colly.HTMLElement) {
			if strings.Contains(e.ChildText("span"), "Киев") {
				companies[e.Request.URL.String()] = companies[e.Request.URL.String()].IncrementPlaces()
			}
		})

		pageCollector.OnRequest(func(r *colly.Request) {
			time.Sleep(5 * time.Second)
			log.Println("Visiting", r.URL)
		})

		pageCollector.OnError(func(r *colly.Response, e error) {
			log.Println("error:", e)
		})

		pageCollector.Visit(listLink + strconv.Itoa(page))
	}

	f := excelize.NewFile()
	index := f.NewSheet("Sheet1")
	f.SetCellValue("Sheet1", "A1", "Company Name")
	f.SetCellValue("Sheet1", "B1", "Work.ua")
	f.SetCellValue("Sheet1", "C1", "Website")
	f.SetCellValue("Sheet1", "D1", "Company Description")
	f.SetCellValue("Sheet1", "E1", "Places")
	counter := 2

	for _, company := range companies {
		if company.placesCount != 0 && company.name != "" && !strings.Contains(company.name, "ФОП") && company.website != "" {
			f.SetCellValue("Sheet1", fmt.Sprintf("A%v", counter), company.name)
			f.SetCellValue("Sheet1", fmt.Sprintf("B%v", counter), company.workUA)
			f.SetCellValue("Sheet1", fmt.Sprintf("C%v", counter), company.website)
			f.SetCellValue("Sheet1", fmt.Sprintf("D%v", counter), company.description)
			f.SetCellValue("Sheet1", fmt.Sprintf("E%v", counter), company.placesCount)
			counter++
		}
	}

	f.SetActiveSheet(index)

	if err := f.SaveAs("Book1.xlsx"); err != nil {
		log.Println(err)
	}
}
