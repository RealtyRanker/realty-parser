// Pass this function to Playwright evaluate on a Domclick search page.
function captureDomclickSearch({ deal, aid, limit }) {
  const prefix = 'window.__SSR_STATE__=';
  const stateScript = [...document.querySelectorAll('script:not([src])')]
    .map((element) => element.textContent.trim()).find((source) => source.startsWith(prefix));
  const source = stateScript?.slice(prefix.length) || '';
  // Read the first object and normalize JavaScript's undefined outside string values.
  let depth = 0;
  let end = 0;
  for (const match of source.matchAll(/"(?:\\.|[^"\\])*"|[{}]/g)) {
    if (match[0] === '{') depth++;
    else if (match[0] === '}' && --depth === 0) {
      end = match.index + 1;
      break;
    }
  }
  if (!end) return { error: document.title };
  const json = source.slice(0, end).replace(/"(?:\\.|[^"\\])*"|\bundefined\b/g,
    (token) => token === 'undefined' ? 'null' : token);
  const state = JSON.parse(json);
  const search = state.search;
  const page = search?.pages?.[search.pagination?.offset];
  if (!page?.ids?.length || state.REGIONS?.aId !== aid) {
    return { error: document.title, region: state.REGIONS?.aId };
  }

  const items = page.ids.map((id) => page.entities[id])
    .filter((offer) => offer?.dealType === deal).slice(0, limit).map((offer) => {
      const { area, floor, rooms, isApartment } = offer.objectInfo || {};
      return {
        id: offer.id,
        path: offer.path,
        dealType: offer.dealType,
        offerType: offer.offerType,
        status: offer.status,
        isSold: offer.isSold,
        price: offer.price,
        isOwner: offer.isOwner,
        objectInfo: { area, floor, rooms, isApartment },
        house: { floors: offer.house?.floors },
        flatComplex: { name: offer.flatComplex?.name },
        location: { lat: offer.location?.lat, lon: offer.location?.lon },
        address: {
          displayName: offer.address?.displayName,
          subways: offer.address?.subways?.map(({ name, remoteness, isUnderConstruction }) =>
            ({ name, remoteness, isUnderConstruction })),
        },
        photos: offer.photos?.slice(0, 3).map(({ url }) => ({ url })),
        updatedDate: offer.updatedDate,
        publishedDate: offer.publishedDate,
      };
    });
  return { result: { items, pagination: { ...search.pagination, total: search.paginationTotal } } };
}
