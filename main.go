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

	xlsxName   string = "companies.xlsx"
	timeLayout string = "02-01-2006 15:04:05"

	workuaMaxPage int = 126
)

// Company структура с данными, которые программа собирает про компанию
type Company struct {
	name        string // название компании
	website     string // веб-сайт компании
	workUA      string // ссылка на work.ua компании
	description string // короткое описание компании
	placesCount int    // количество вакансий в Киеве
	used        bool
}

// IncrementPlaces добавляет 1 к счетчику вакансий
func (c Company) IncrementPlaces() Company {
	c.placesCount++
	return c
}

// NewCompany функция-генератор новой компании
func NewCompany(name, website, workUA, descr string, places int, used bool) Company {
	if !used {
		log.Printf("creating new company: %v\n", name)
	}
	return Company{
		name:        name,
		website:     website,
		workUA:      workUA,
		description: descr,
		placesCount: places,
		used:        used,
	}
}

// загружает в карту уже спаршенные компании
func loadCompanies(companies map[string]Company) {

	f, err := excelize.OpenFile(xlsxName)
	if err != nil {
		log.Println(err)
	}

	log.Println("loading companies..")

	sheetList := f.GetSheetList()

	for _, sheet := range sheetList {
		for i := 2; ; i++ {
			name, err := f.GetCellValue(sheet, fmt.Sprintf("A%v", i))
			if err != nil {
				log.Println(err)
			}

			workUA, err := f.GetCellValue(sheet, fmt.Sprintf("B%v", i))
			if err != nil {
				log.Println(err)
			}

			website, err := f.GetCellValue(sheet, fmt.Sprintf("C%v", i))
			if err != nil {
				log.Println(err)
			}

			description, err := f.GetCellValue(sheet, fmt.Sprintf("D%v", i))
			if err != nil {
				log.Println(err)
			}

			placesCount, err := f.GetCellValue(sheet, fmt.Sprintf("E%v", i))
			if err != nil {
				log.Println(err)
			}

			if name == "" || website == "" || workUA == "" || description == "" || placesCount == "" {
				log.Println("loading complete")
				break
			}

			placesCountInt, err := strconv.Atoi(placesCount)
			if err != nil {
				log.Println(err)
			}

			companies[workUA] = NewCompany(
				name,
				website,
				workUA,
				description,
				placesCountInt,
				true)
		}
	}
}

func saveCompanies(companies map[string]Company) {
	f, err := excelize.OpenFile(xlsxName)
	if err != nil {
		log.Println(err)
	}

	// название листа генерируется из текущего времени
	sheetName := time.Now().Format(timeLayout)

	index := f.NewSheet(sheetName)

	f.SetCellValue(sheetName, "A1", "Company Name")
	f.SetCellValue(sheetName, "B1", "Work.ua")
	f.SetCellValue(sheetName, "C1", "Website")
	f.SetCellValue(sheetName, "D1", "Company Description")
	f.SetCellValue(sheetName, "E1", "Places")

	counter := 2
	for _, company := range companies {

		if company.placesCount != 0 && company.name != "" && !strings.Contains(company.name, "ФОП") && company.website != "" && !company.used {

			f.SetCellValue(sheetName, fmt.Sprintf("A%v", counter), company.name)
			f.SetCellValue(sheetName, fmt.Sprintf("B%v", counter), company.workUA)
			f.SetCellValue(sheetName, fmt.Sprintf("C%v", counter), company.website)
			f.SetCellValue(sheetName, fmt.Sprintf("D%v", counter), company.description)
			f.SetCellValue(sheetName, fmt.Sprintf("E%v", counter), company.placesCount)

			counter++
		}

	}

	f.SetActiveSheet(index)

	if err := f.SaveAs(xlsxName); err != nil {
		log.Println(err)
	}

}

func parseCompanies(companies map[string]Company) {
	for page := 1; page <= workuaMaxPage; page++ {
		log.Printf("parsing new page: #%v\n", page)

		// pageCollector собирает с страницы-списка ссылки на компании
		pageCollector := colly.NewCollector()

		// companyCollector собирает со страницы компании всю информацию
		companyCollector := colly.NewCollector()

		// cityCollector собирает информацию про доступные вакансии и города
		cityCollector := colly.NewCollector()

		companyCollector.OnXML("/html/body/section/div/div/div[1]/div[2]", func(e *colly.XMLElement) {

			var flag bool = true

			for company := range companies {
				if companies[company].name == e.ChildText("div/h1") {
					log.Printf("found same company: %v\n", companies[company].name)
					flag = false
				}
			}

			if flag {
				companies[e.Request.URL.String()] = NewCompany(
					e.ChildText("div/h1"),
					e.ChildAttr("div/div/div/p/span/a", "href"),
					e.Request.URL.String(),
					fmt.Sprintf("%v %v", e.ChildText("div/p[1]"), e.ChildText("div/p[2]")),
					0,
					false)
			}
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
			log.Println("visiting", r.URL)
		})

		pageCollector.OnError(func(r *colly.Response, e error) {
			log.Println("error:", e)
		})

		pageCollector.Visit(listLink + strconv.Itoa(page))
	}

}

func main() {
	// companies карта в которой будут хранитья компании
	// в роли ключа выступает ссылка на work.ua
	var companies map[string]Company = map[string]Company{}
	for {
		fmt.Println("Добро пожаловать в COMPANY-PARSER 3000-EXTREME ULTRA\n\nТребования к использованию:\n\t - подключение к сети Интернет\n\t - наличие документа companies.xlsx в директории с программой\n\nПри возникновении любых ошибок писать в tg - @adgvkty\n©2021 Adgvkty Inc\n\nВы согласились и узнали? (Y/N):")
		var response string
		fmt.Scanf("%s", &response)
		if response == "Y" || response == "y" {
			log.Println("starting...")
			loadCompanies(companies)
			parseCompanies(companies)
			saveCompanies(companies)
			return
		} else if response == "N" {
			return
		}
	}
}
