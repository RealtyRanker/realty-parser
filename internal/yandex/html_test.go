package yandex

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestParseSavedSearch(t *testing.T) {
	page, err := ParseSavedPage(strings.NewReader(`<main>
  <div class="OffersSerpItem__generalInfoInnerContainer"><a href="/offer/9223372036854775808/?from=serp">Квартира</a></div>
  <div class="OffersSerpItem__generalInfoInnerContainer"><a href="https://realty.yandex.ru/offer/9223372036854775808/">Квартира</a></div>
  <div class="OffersSerpItem__generalInfoInnerContainer"><a href="https://example.com/offer/2/">Чужая ссылка</a></div>
  <a href="/offer/3/">Недавно просмотрено</a>
  <a aria-label="Следующая страница" href="/moskva/kupit/kvartira/?page=3&amp;sort=DATE_DESC">Вперёд</a>
</main>`), 1)
	if err != nil {
		t.Fatal(err)
	}
	want := &SearchPage{
		ListingURLs: []string{"https://realty.yandex.ru/offer/9223372036854775808/"},
		NextURL:     "https://realty.yandex.ru/moskva/kupit/kvartira/?page=3&sort=DATE_DESC",
	}
	if !reflect.DeepEqual(page.Search, want) || page.Listing != nil {
		t.Fatalf("got %+v, want %+v", page.Search, want)
	}
}

const savedRentHTML = `<link rel="canonical" href="https://realty.yandex.ru/offer/7667794382545778816/">
<script type="application/ld+json">{"@graph":[
  {"@type":"Product","sku":"7667794382545778816","category":"APARTMENT","description":"Описание квартиры.","image":["//avatars.mds.yandex.net/get-realty-offers/123/test/app_large","//avatars.mds.yandex.net/get-realty-offers/123/test/large"]},
  {"@type":"Apartment","url":"https://realty.yandex.ru/offer/7667794382545778816","floorLevel":"9","floorSize":{"value":37.9},"numberOfRooms":1,"geo":{"latitude":55.71536,"longitude":37.880302}},
  {"@type":"Offer","url":"https://realty.yandex.ru/offer/7667794382545778816","price":55000,"priceCurrency":"RUB","availability":"https://schema.org/InStock"}
]}</script>
<main>
  <nav class="OfferCard__breadcrumbs--1VcHz"><a href="/moskva/snyat/">Снять</a></nav>
  <div class="OfferCard__offerCardSummary--3PX9E">
    <h1>37,9 м², 1-комнатная квартира</h1>
    <div class="OfferCardSummaryInfo__price--20T_Q">55&nbsp;000 ₽&nbsp;в&nbsp;месяц</div>
    <div class="OfferCardSummary__location--22FMW">
      <span class="CardLocation__addressItem--k8XSk">Москва, улица Руднёвка, 3</span>
      <div class="MetroWithTime"><span class="MetroWithTime__title">Улица Дмитриевского</span><span class="MetroWithTime__time">7 мин.</span><svg class="Icon Icon_type_small-pedestrian"></svg></div>
    </div>
    <ul>
      <li class="OfferCardCheck__row--20GSJ"><span class="OfferCardCheck__rowKey--254nq">Залог</span><span class="OfferCardCheck__rowValue--jS4Vx">25&nbsp;000 ₽</span></li>
      <li class="OfferCardCheck__row--20GSJ"><span class="OfferCardCheck__rowKey--254nq">Комиссия агенту<span class="OfferCardCheck__badge--30i04">0%</span></span><span class="OfferCardCheck__rowValue--jS4Vx">0 ₽</span></li>
      <li class="OfferCardCheck__row--20GSJ"><span class="OfferCardCheck__rowKey--254nq">Коммунальные услуги</span><span class="OfferCardCheck__rowValue--jS4Vx">включены в стоимость</span></li>
      <li class="OfferCardCheck__row--20GSJ"><span class="OfferCardCheck__rowKey--254nq">Итого</span><span class="OfferCardCheck__rowValue--jS4Vx">80&nbsp;000 ₽</span></li>
    </ul>
  </div>
  <div class="OfferCardHighlight__container--1Fsba"><span class="OfferCardHighlight__value--XLPsC">20 м²</span><span class="OfferCardHighlight__label--GOzzA">жилая</span></div>
  <div class="OfferCardHighlight__container--1Fsba"><span class="OfferCardHighlight__value--XLPsC">7 м²</span><span class="OfferCardHighlight__label--GOzzA">кухня</span></div>
  <div class="OfferCardHighlight__container--1Fsba"><span class="OfferCardHighlight__value--XLPsC">9 этаж</span><span class="OfferCardHighlight__label--GOzzA">из 12</span></div>
  <div class="OfferCardDetailsFeatures__feature--Ojfho">Посудомойка</div>
  <div class="OfferCardDetailsFeatures__feature--Ojfho">Без детей</div>
  <div class="OfferCardBuildingFeatures__feature--31MKF">179 квартир</div>
</main>`

func TestParseSavedRentalListing(t *testing.T) {
	page, err := ParseSavedPage(strings.NewReader(savedRentHTML), 1)
	if err != nil {
		t.Fatal(err)
	}
	l := page.Listing
	f := l.Flat
	if f.Price != 55000 || f.DealType != "rent" || f.Deposit != 25000 || f.Comission != 0 || l.Rent.Commission == nil ||
		l.Rent.InitialPayment == nil || *l.Rent.InitialPayment != 80000 || l.Rent.Utilities != "включены в стоимость" {
		t.Fatalf("unexpected rental terms: %+v %+v", f, l.Rent)
	}
	if f.RoomNumber != 1 || f.TotalArea != 37.9 || f.LivingArea != 20 || f.KitchenArea != 7 || f.Floor != 9 || f.MaxFloor != 12 ||
		f.Latitude != 55.71536 || f.BuildingApartmentsNumber != 179 || !f.HasDishwasher || f.ChildrenAllowed ||
		!reflect.DeepEqual(f.UndergroundStations, []string{"Улица Дмитриевского"}) || l.Description == "" {
		t.Fatalf("unexpected apartment: %+v", f)
	}
	if len(f.PhotoURLs) != 1 || !strings.HasPrefix(f.PhotoURLs[0], "https://") {
		t.Fatalf("unexpected photo URLs: %v", f.PhotoURLs)
	}
}

func TestParseSavedPageRejectsUnsupportedInputs(t *testing.T) {
	if _, err := ParseSavedPage(strings.NewReader(`<title>Вы не робот?</title>`), 1); !errors.Is(err, ErrBlocked) {
		t.Fatalf("expected ErrBlocked, got %v", err)
	}
	daily := strings.Replace(savedRentHTML, "в&nbsp;месяц", "в&nbsp;сутки", 1)
	if _, err := ParseSavedPage(strings.NewReader(daily), 1); err == nil {
		t.Fatal("accepted a daily rental")
	}
}
