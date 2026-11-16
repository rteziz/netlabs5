const russianTypes = new Map([
    ['bus_stop', 'Автобусная остановка'],
    ['platform', 'Платформа'],
    ['station', 'Станция'],
    ['tram_stop', 'Трамвайная остановка'],
    ['subway_entrance', 'Вход в метро'],
    ['residential', 'Жилая улица'],
    ['unclassified', 'Неопознанное место'],
    ['pedestrian', 'Пешеходная улица'],
    ['footway', 'Пешеходная дорожка'],
    ['primary', 'Главная дорога'],
    ['secondary', 'Второстепенная дорога'],
    ['tertiary', 'Дорога третьего класса'],
    ['museum', 'Музей'],
    ['attraction', 'Достопримечательность'],
    ['viewpoint', 'Смотровая площадка'],
    ['artwork', 'Произведение искусства'],
    ['hotel', 'Гостиница'],
    ['hostel', 'Хостел'],
    ['park', 'Парк'],
    ['garden', 'Сад'],
    ['square', 'Площадь'],
    ['playground', 'Детская площадка'],
    ['stadium', 'Стадион'],
    ['sports_centre', 'Спортивный центр'],
    ['cafe', 'Кафе'],
    ['restaurant', 'Ресторан'],
    ['fast_food', 'Быстрое питание'],
    ['bar', 'Бар'],
    ['pub', 'Паб'],
    ['school', 'Школа'],
    ['university', 'Университет'],
    ['hospital', 'Больница'],
    ['pharmacy', 'Аптека'],
    ['library', 'Библиотека'],
    ['cinema', 'Кинотеатр'],
    ['theatre', 'Театр'],
    ['parking', 'Парковка'],
    ['supermarket', 'Супермаркет'],
    ['mall', 'Торговый центр'],
    ['monument', 'Памятник'],
    ['memorial', 'Мемориал'],
    ['castle', 'Замок'],
    ['church', 'Церковь'],
]);
let availableTypes = [];
// Возвращает подпись типа места для текущего языка.
function typeLabel(code) {
    return language === 'ru' ? russianTypes.get(code) || code : code;
}
// Переводит введённое название типа в OSM-код.
function typeCode(value) {
    const input = value.trim().toLowerCase();
    for (const [code, label] of russianTypes) {
        if (label.toLowerCase() === input)
            return code;
    }
    return input;
}
// Заполняет подсказки поля типа места.
function renderTypes() {
    const list = $('place-types');
    list.replaceChildren();
    for (const code of availableTypes) {
        const option = document.createElement('option');
        option.value = typeLabel(code);
        list.append(option);
    }
}
'use strict';
const messages = {
    ru: {
        placeType: 'Тип места (необязательно)', typeHint: 'Например, «Автобусная остановка». Пустое поле — обычный поиск.', source: 'Источник мест', fallback: 'Wikipedia (OpenTripMap недоступен или не нашёл мест)', brand: 'Места', language: 'Язык', eyebrow: 'НЕБОЛЬШОЙ ПОВОД ВЫЙТИ ИЗ ДОМА', headline: 'У каждого места\nесть история.', intro: 'Найдите точку на карте. Узнайте, что вокруг и какую погоду взять с собой.',
        pauseMotion: 'Пауза фона', resumeMotion: 'Включить фон', searchTitle: 'Поиск места', queryLabel: 'Город, улица или место', placeholder: 'Например, Куртамыш или Новосибирск', search: 'Найти', radius: 'Радиус, м', limit: 'Мест в отчёте', hint: 'Адреса · погода · истории поблизости', cancel: 'Отменить', retry: 'Попробовать снова', errorTitle: 'Не получилось загрузить данные', choose: 'Результаты', emptyTitle: 'Начните с любопытства', emptyText: 'Введите название выше — мы найдём варианты, а вы выберете, куда отправиться.', footer: 'Для коротких прогулок и больших открытий.',
        searching: 'Ищем варианты адреса…', exploring: 'Загрузка…', canceled: 'Поиск отменён.', noLocations: 'Ничего не найдено. Уточните название или добавьте город.', found: '', ready: '', reportLabel: 'Место', temperature: 'Сейчас', feels: 'Ощущается как', wind: 'Ветер', speed: 'м/с', pressure: 'Давление на уровне моря', humidity: 'Влажность', mmHg: 'мм рт. ст.', photo: 'Фото: автор и лицензия ↗', meter: 'м', localTime: 'Местное время', nearby: 'Что интересного рядом', article: 'Читать в Wikipedia ↗', noPlaces: 'Мест поблизости не найдено. Увеличьте радиус.', cold: 'Режим пельменя в морозилке: одевайтесь теплее.', licensing: 'Погода: Open-Meteo (CC BY 4.0). Описания: Wikipedia; авторство и лицензии фотографий — по ссылкам под ними.',
        errors: {
            missing_key: 'На сервере задайте GRAPHHOPPER_API_KEY и перезапустите приложение.', auth: 'API отклонил ключ или доступ. Проверьте ключи и их активацию.', rate_limit: 'Сервис временно ограничил запросы (HTTP 429). Подождите немного и попробуйте снова.', timeout: 'Сервисы не успели ответить. Попробуйте ещё раз чуть позже.', canceled: 'Запрос отменён.', busy: 'Слишком много активных операций. Попробуйте через пару минут.', expired: 'Операция уже завершена или устарела. Запустите поиск снова.', invalid_query: 'Введите название места (до 150 символов).', invalid_location: 'Не удалось прочитать координаты. Выберите локацию заново.', invalid_request: 'Проверьте параметры: радиус 10–10000 м, число мест 1–10.', upstream: 'Один из сервисов сейчас недоступен. Попробуйте повторить запрос.', network: 'Нет связи с приложением. Проверьте, что Go-сервер запущен.'
        }
    },
    en: {
        placeType: 'Place type (optional)', typeHint: 'For example, bus_stop. Leave empty for a regular search.', source: 'Places source', fallback: 'Wikipedia (OpenTripMap unavailable or no results)', brand: 'Places', language: 'Language', eyebrow: 'A LITTLE REASON TO STEP OUTSIDE', headline: 'Every place\nhas a story.', intro: 'Find a point on the map. Discover what’s nearby and what the weather has in store.',
        pauseMotion: 'Pause background', resumeMotion: 'Animate background', searchTitle: 'Find a place', queryLabel: 'City, street or place', placeholder: 'Try Kichijoji, Berlin or Novosibirsk', search: 'Search', radius: 'Radius, m', limit: 'Places to show', hint: 'Addresses · weather · nearby stories', cancel: 'Cancel', retry: 'Try again', errorTitle: 'Could not load the details', choose: 'Results', emptyTitle: 'Start with curiosity', emptyText: 'Enter a name above. We’ll find the options, and you choose where to go.', footer: 'For short walks and big discoveries.',
        searching: 'Looking for matching addresses…', exploring: 'Loading…', canceled: 'Search canceled.', noLocations: 'Nothing found. Try a more specific name or add a city.', found: '', ready: '', reportLabel: 'Place', temperature: 'Right now', feels: 'Feels like', wind: 'Wind', speed: 'm/s', pressure: 'Sea-level pressure', humidity: 'Humidity', mmHg: 'mmHg', photo: 'Photo: author and license ↗', meter: 'm', localTime: 'Local time', nearby: 'Interesting places nearby', article: 'Read on Wikipedia ↗', noPlaces: 'No places found nearby. Increase the radius.', cold: 'Frozen dumpling mode: dress warmly.', licensing: 'Weather: Open-Meteo (CC BY 4.0). Descriptions: Wikipedia; photo credits and licenses are linked below each image.',
        errors: {
            missing_key: 'Set GRAPHHOPPER_API_KEY on the server and restart the app.', auth: 'API key or access rejected. Check your keys and their activation.', rate_limit: 'The service temporarily limited requests (HTTP 429). Please wait a little and try again.', timeout: 'The services took too long to respond. Please try again later.', canceled: 'Request canceled.', busy: 'Too many active operations. Please try again in a couple of minutes.', expired: 'This operation has expired. Please start a new search.', invalid_query: 'Enter a place name (up to 150 characters).', invalid_location: 'Could not read the coordinates. Please select a location again.', invalid_request: 'Check your settings: radius 10–10000 m, 1–10 places.', upstream: 'One of the services is unavailable. Please try again later.', network: 'Cannot reach the app. Make sure the Go server is running.'
        }
    }
};
const $ = id => document.getElementById(id);
const pollingInterval = 400;
let language = 'ru', generation = 0, current = null, lastAction = null, configured = false;
// Возвращает перевод по ключу.
const t = key => messages[language][key];
// Создаёт HTML-элемент с классом и текстом.
function element(tag, className, text) {
    const node = document.createElement(tag);
    if (className)
        node.className = className;
    if (text !== undefined)
        node.textContent = text;
    return node;
}
// Обновляет подписи и подсказки при смене языка.
function translate() {
    const field = $('place-type');
    field.placeholder = ['bus_stop', 'museum', 'park'].map(typeLabel).join(', ');
    field.value = typeLabel(typeCode(field.value));
    renderTypes();
    document.documentElement.lang = language;
    document.title = t('brand');
    document.querySelectorAll('[data-i18n]').forEach(node => {
        node.textContent = t(node.dataset.i18n);
    });
    $('query').placeholder = t('placeholder');
}
// Собирает параметры запроса из полей формы.
function options() {
    return {
        language, placeType: typeCode($('place-type').value), radius: Number($('radius').value), limit: Number($('limit').value)
    };
}
// Очищает область результатов предыдущего поиска.
function clearResults() {
    $('locations').hidden = true;
    $('report').hidden = true;
    $('report').replaceChildren();
    $('error-panel').hidden = true;
    $('empty-state').hidden = false;
}
// Удаляет задачу на сервере по её идентификатору.
function release(id) {
    if (id)
        fetch('/api/jobs/' + encodeURIComponent(id), {
            method: 'DELETE', keepalive: true
        }).catch(() => {
        });
}
// Останавливает текущий опрос и освобождает задачу.
function cancel(silent = false) {
    generation++;
    if (current) {
        current.controller.abort();
        release(current.id);
        current = null;
    }
    $('cancel').hidden = true;
    $('search-button').disabled = !configured;
    if (!silent)
        $('status').textContent = t('canceled');
}
// Показывает сообщение по коду ошибки.
function showError(code) {
    $('error-text').textContent = messages[language].errors[code] || messages[language].errors.upstream;
    $('error-panel').hidden = false;
}
// Выдерживает паузу между опросами с поддержкой отмены.
function pause(signal) {
    return new Promise((resolve, reject) => {
        // Прерывает ожидание и снимает таймер.
        const abort = () => {
            clearTimeout(timer);
            reject(new DOMException('Canceled', 'AbortError'));
        };
        const timer = setTimeout(() => {
            signal.removeEventListener('abort', abort);
            resolve();
        }, pollingInterval);
        if (signal.aborted)
            abort();
        else
            signal.addEventListener('abort', abort, {
                once: true
            });
    });
}
// Читает JSON и проверяет статус HTTP-ответа.
async function json(response) {
    let body;
    try {
        body = await response.json();
    }
    catch {
        throw new Error('network');
    }
    if (!response.ok)
        throw new Error(body.error || 'upstream');
    return body;
}
// Запускает задачу и опрашивает сервер до получения результата.
async function execute(kind, payload) {
    cancel(true);
    const token = generation;
    const operation = {
        id: null, controller: new AbortController()
    };
    current = operation;
    lastAction = {
        kind, payload
    };
    $('error-panel').hidden = true;
    $('report').hidden = true;
    $('empty-state').hidden = true;
    $('locations').hidden = true;
    $('cancel').hidden = false;
    $('search-button').disabled = true;
    $('status').textContent = t(kind === 'search' ? 'searching' : 'exploring');
    try {
        // Do not abort creation: if canceled while POST is in flight, delete the returned job.
        const created = await json(await fetch('/api/' + kind, {
            method: 'POST', headers: {
                'Content-Type': 'application/json'
            }, body: JSON.stringify(payload)
        }));
        operation.id = created.id;
        if (token !== generation) {
            release(created.id);
            return;
        }
        while (token === generation) {
            const job = await json(await fetch('/api/jobs/' + encodeURIComponent(created.id), {
                signal: operation.controller.signal
            }));
            if (token !== generation)
                return;
            if (job.status === 'error')
                throw new Error(job.error);
            if (job.status === 'done') {
                if (kind === 'search')
                    renderLocations(job.result || []);
                else
                    renderReport(job.result);
                return;
            }
            await pause(operation.controller.signal);
        }
    }
    catch (error) {
        if (token === generation && error.name !== 'AbortError') {
            $('status').textContent = '';
            showError(error.message);
        }
    }
    finally {
        release(operation.id);
        if (token === generation) {
            current = null;
            $('cancel').hidden = true;
            $('search-button').disabled = !configured;
        }
    }
}
// Показывает найденные адреса для выбора.
function renderLocations(locations) {
    const list = $('location-list');
    list.replaceChildren();
    $('status').textContent = t(locations.length ? 'found' : 'noLocations');
    $('locations').hidden = !locations.length;
    locations.forEach((location, index) => {
        const button = element('button', 'location-button');
        button.type = 'button';
        const details = element('span');
        details.append(element('span', 'location-name', location.Name), element('span', 'coords', `${location.Latitude.toFixed(5)}, ${location.Longitude.toFixed(5)}`));
        if (location.OSMValue)
            details.append(element('span', 'coords', language === 'ru' ? typeLabel(location.OSMValue) : location.OSMKey + ' = ' + location.OSMValue));
        button.append(element('span', 'index', String(index + 1).padStart(2, '0')), details, element('span', 'arrow', '↗'));
        button.addEventListener('click', () => {
            if ($('search-form').reportValidity())
                execute('explore', {
                    ...options(), location
                });
        });
        list.append(button);
    });
    $('location-count').textContent = String(locations.length).padStart(2, '0');
}
// Показывает погоду и карточки мест из готового отчёта.
function renderReport(result) {
    const fragment = document.createDocumentFragment();
    const header = element('div', 'report-header');
    header.append(element('p', 'eyebrow', t('reportLabel')), element('h2', '', result.Location.Name));
    header.lastChild.id = 'report-title';
    header.append(element('p', 'coords', `${result.Location.Latitude.toFixed(5)}, ${result.Location.Longitude.toFixed(5)}`));
    fragment.append(header);
    if (result.PlacesSource)
        fragment.append(element('p', 'report-note', t('source') + ': ' + (result.PlacesSource.includes('fallback') ? t('fallback') : result.PlacesSource)));
    const weather = element('div', 'weather');
    // Создаёт элемент для одного погодного показателя.
    const measure = (label, value, main = false) => {
        const box = element('div');
        box.append(element('div', 'weather-label', label), element('div', main ? 'temperature' : 'weather-value', value));
        return box;
    };
    // Форматирует число с учётом языка интерфейса.
    const number = value => Number(value).toLocaleString(language, {
        maximumFractionDigits: 1
    });
    weather.append(measure(t('temperature'), number(result.Weather.temperature) + ' °C', true), measure(t('feels'), number(result.Weather.feels_like) + ' °C'), measure(t('wind'), number(result.Weather.wind_speed) + ' ' + t('speed')));
    const mmHgPerHpa = 0.75006168;
    const pressureMmHg = result.Weather.pressure * mmHgPerHpa;
    weather.append(measure(t('pressure'), number(pressureMmHg) + ' ' + t('mmHg')), measure(t('humidity'), number(result.Weather.humidity) + ' %'));
    if (result.Weather.description)
        weather.append(element('div', 'weather-description', result.Weather.description));
    const time = result.Weather.time.replace('T', ' · ');
    weather.append(element('div', 'weather-time', `${t('localTime')}: ${time}`));
    fragment.append(weather);
    const places = result.Places || [];
    const heading = element('div', 'section-heading');
    heading.append(element('h2', '', t('nearby')), element('span', 'badge', String(places.length).padStart(2, '0')));
    fragment.append(heading);
    const grid = element('div', 'places-grid');
    places.forEach((place, index) => {
        const card = element('article', 'place');
        const top = element('div', 'place-top');
        top.append(element('span', '', String(index + 1).padStart(2, '0')), element('span', '', `${Math.round(place.Distance)} ${t('meter')}`));
        if (place.ImageURL && place.ImagePageURL) {
            try {
                const source = new URL(place.ImageURL), credit = new URL(place.ImagePageURL);
                if (source.protocol === 'https:' && source.hostname === 'upload.wikimedia.org' && credit.protocol === 'https:' && /(^|\.)wikipedia\.org$/.test(credit.hostname)) {
                    const photo = element('img', 'place-image');
                    photo.src = source.href;
                    photo.alt = place.Title;
                    photo.loading = 'lazy';
                    photo.addEventListener('error', () => photo.remove(), {
                        once: true
                    });
                    const attribution = element('a', 'photo-credit', t('photo'));
                    attribution.href = credit.href;
                    attribution.target = '_blank';
                    attribution.rel = 'noopener noreferrer';
                    card.append(photo, attribution);
                }
            }
            catch {
            }
        }
        card.append(top, element('h3', '', place.Title), element('p', '', place.Description));
        if (place.URL) {
            const link = element('a', '', t('article'));
            const url = new URL(place.URL);
            if (url.protocol === 'https:' && /(^|\.)wikipedia\.org$/.test(url.hostname)) {
                link.href = url.href;
                link.target = '_blank';
                link.rel = 'noopener noreferrer';
                card.append(link);
            }
        }
        grid.append(card);
    });
    fragment.append(places.length ? grid : element('p', 'intro', t('noPlaces')), element('p', 'report-note', t('licensing')));
    // Commit the complete report in one DOM update after all APIs have finished.
    $('report').replaceChildren(fragment);
    $('report').hidden = false;
    $('status').textContent = t('ready');
    $('report').focus({
        preventScroll: true
    });
    $('report').scrollIntoView({
        behavior: 'smooth', block: 'start'
    });
}
$('search-form').addEventListener('submit', event => {
    event.preventDefault();
    if (configured)
        execute('search', {
            ...options(), query: $('query').value.trim()
        });
});
$('cancel').addEventListener('click', () => cancel());
$('retry').addEventListener('click', () => {
    if (lastAction)
        execute(lastAction.kind, lastAction.payload);
    else
        window.location.reload();
});
$('language').addEventListener('change', () => {
    cancel(true);
    language = $('language').value;
    translate();
    lastAction = null;
    clearResults();
    $('status').textContent = '';
});
for (const id of ['radius', 'limit', 'place-type'])
    $(id).addEventListener('change', () => {
        cancel(true);
        lastAction = null;
        clearResults();
        $('status').textContent = '';
    });
window.addEventListener('pagehide', () => cancel(true));
// Загружает начальные настройки формы с сервера.
async function initialize() {
    $('search-button').disabled = true;
    try {
        const config = await json(await fetch('/api/config'));
        language = config.language;
        $('language').value = language;
        $('radius').value = config.radius;
        $('limit').value = config.limit;
        configured = true;
    }
    catch {
        showError('network');
    }
    translate();
    $('search-button').disabled = !configured;
}
initialize();
fetch('/api/place-types').then(json).then(types => {
    availableTypes = types;
    renderTypes();
}).catch(() => {
});
