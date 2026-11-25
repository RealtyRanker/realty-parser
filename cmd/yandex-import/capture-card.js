// Run this function with Playwright evaluate on an open Yandex Realty listing.
function captureYandexCard() {
  const main = document.querySelector('main');
  const one = (root, name) => root?.querySelector(`[class*="${name}--"]`);
  const all = (root, name) => [...(root?.querySelectorAll(`[class*="${name}--"]`) || [])];
  const text = (node) => node?.textContent.replace(/\s+/g, ' ').trim() || '';
  const escape = (value) => String(value).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/"/g, '&quot;');
  const wrap = (name, content) => `<div class="${name}">${content}</div>`;
  const field = (root, name) => {
    const value = text(one(root, name));
    return value ? wrap(name, escape(value)) : '';
  };
  const fields = (root, name) => all(root, name).map((node) => wrap(name, escape(text(node)))).join('');

  const card = one(main, 'OfferCard__offerCardSummary');
  const title = text(card?.querySelector('h1'));
  const url = document.querySelector('link[rel="canonical"]')?.getAttribute('href') || '';
  if (!title || !/^https:\/\/realty\.yandex\.ru\/offer\/\d+\/?$/.test(url)) {
    return { title: document.title, html: '' };
  }

  const graph = [];
  for (const element of document.querySelectorAll('script[type="application/ld+json"]')) {
    let nodes;
    try {
      nodes = JSON.parse(element.textContent)['@graph'];
    } catch {
      continue;
    }
    for (const item of nodes || []) {
      const type = item['@type'];
      if (type === 'Product') {
        const { sku, category, image } = item;
        graph.push({ '@type': type, sku, category, image: Array.isArray(image) ? image.slice(0, 1) : [] });
      } else if (type === 'Apartment') {
        const { url, numberOfRooms, floorSize, floorLevel, geo, address } = item;
        graph.push({ '@type': type, url, numberOfRooms, floorSize, floorLevel, geo, address });
      } else if (type === 'Offer') {
        const { url, price, priceCurrency, availability } = item;
        graph.push({ '@type': type, url, price, priceCurrency, availability });
      }
    }
  }

  const trail = one(main, 'OfferCard__breadcrumbs');
  const breadcrumbs = [...(trail?.querySelectorAll('a') || [])]
    .filter((link) => /^\/(?!\/).*(?:kupit|snyat)\//.test(link.getAttribute('href') || ''))
    .map((link) => `<a href="${escape(link.getAttribute('href'))}">${escape(text(link))}</a>`).join('');
  const location = one(card, 'OfferCardSummary__location');
  const transit = [...(location?.querySelectorAll('.MetroWithTime') || [])].map((stop) => {
    const name = text(stop.querySelector('.MetroWithTime__title'));
    const time = text(stop.querySelector('.MetroWithTime__time'));
    const icon = stop.querySelector('[class*="Icon_type_small-"]')?.getAttribute('class') || '';
    return wrap('MetroWithTime', wrap('MetroWithTime__title', escape(name)) +
      wrap('MetroWithTime__time', escape(time)) + `<i class="${escape(icon)}"></i>`);
  }).join('');
  const rows = all(card, 'OfferCardCheck__row').map((row) => wrap('OfferCardCheck__row',
    field(row, 'OfferCardCheck__rowKey') + field(row, 'OfferCardCheck__rowValue') + field(row, 'OfferCardCheck__badge'))).join('');
  const highlights = all(main, 'OfferCardHighlight__container').map((item) => wrap('OfferCardHighlight__container',
    field(item, 'OfferCardHighlight__value') + field(item, 'OfferCardHighlight__label'))).join('');
  const gallery = one(main, 'OfferCard__gallery');
  const photos = [...(gallery?.querySelectorAll('img[class*="GalleryV2Slide__wrappedImage--"]') || [])]
    .map((node) => node.getAttribute('src') || '')
    .filter((src) => /^(https:)?\/\/avatars\.mds\.yandex\.net\//.test(src))
    .map((src) => `<img class="GalleryV2Slide__wrappedImage" src="${escape(src)}">`).join('');
  const summary = `<h1>${escape(title)}</h1>` +
    ['OfferCardSummaryInfo__price', 'OfferCardSummaryHeader__text', 'OfferCardAuthorBadge__category']
      .map((name) => field(card, name)).join('') + fields(card, 'OfferCardSummaryTags__badgeText') + rows +
    wrap('OfferCardSummary__location', field(location, 'CardLocation__addressItem') + transit);
  const schema = JSON.stringify({ '@graph': graph }).replace(/</g, '\\u003c');

  return {
    title,
    url,
    html: `<link rel="canonical" href="${url}"><script type="application/ld+json">${schema}</script><main>` +
      wrap('OfferCard__breadcrumbs', breadcrumbs) + wrap('OfferCard__gallery', photos) +
      wrap('OfferCard__offerCardSummary', summary) + highlights + fields(main, 'OfferCardDetailsFeatures__feature') +
      fields(main, 'OfferCardBuildingFeatures__feature') + '</main>',
  };
}
