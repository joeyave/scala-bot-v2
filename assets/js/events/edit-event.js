import InstantSearch from "./../../instant_search/js/InstantSearch.js";

(function () {
    Telegram.WebApp.expand()

    let searchOverlayElement = document.getElementById("search-overlay")
    searchOverlayElement.onclick = (e) => {
        if (e.target !== searchOverlayElement) return;
        searchOverlayElement.style.display = "none"
    }

    const songsElement = document.getElementById("songs");

    new Sortable(songsElement, {
        // handle: ".item",  // Drag handle selector within list items
        // draggable: ".item",  // Specifies which items inside the element should be draggable
        delay: 150,
        delayOnTouchOnly: true,
        animation: 0,
        onUpdate: function (/**Event*/evt) {
            Telegram.WebApp.MainButton.show()
        },
    });

    const searchElement = document.querySelector("#search");
    new InstantSearch(searchElement, {
        searchUrl: new URL("/api/drive-files/search", window.location.origin),
        queryParam: "q",
        responseParser: (responseData) => {
            return responseData.results;
        },
        templateFunction: (result) => {
            return `
            <div class="instant-search__title">${result.name}</div>
<!--            <p class="instant-search__paragraph">${result.occupation}</p>-->
        `;
        },
        resultEventListener: (result) => {
            return [
                "click", async () => {
                    let resp = await fetch(`/api/songs/find-by-drive-file-id?driveFileId=${result.id}`, {
                        method: "get",
                        headers: {'Content-Type': 'application/json'},
                    })

                    let data = await resp.json()

                    console.log(data)

                    Telegram.WebApp.MainButton.show()
                    searchOverlayElement.style.display = "none";

                    let songs = document.getElementById("songs").getElementsByTagName("span")
                    console.log(songs)

                    let exists = false
                    for (let i = 0; i < songs.length; i++) {
                        let songId = songs[i].getAttribute("data-song-id")
                        if (songId === data.song.id) {
                            exists = true
                            break
                        }
                    }
                    if (!exists) {
                        songsElement.insertAdjacentHTML("afterbegin",
                            `<div class="item">
                            <span class="text" data-song-id=${data.song.id}>${result.name}</span>
                            <i id="delete-song-icon" class="fas fa-trash-alt"></i>
                        </div>`
                        );
                    }
                }
            ]
        }
    });

    let addSongButton = document.getElementById("add-song-button")
    addSongButton.onclick = () => {
        searchOverlayElement.style.display = "block"
        document.getElementById("song-search-input").focus()
    }

    document.addEventListener("click", (e) => {
        if (e.target.id === "delete-song-icon") {
            e.target.parentElement.remove()
            Telegram.WebApp.MainButton.show()
        }
    })

    let form = document.getElementById('event-form');

    form.addEventListener('input', function (event) {
        Telegram.WebApp.MainButton.show()
    })

    Telegram.WebApp.ready()

    let name = document.getElementById("name")
    let date = document.getElementById("date")
    let notes = document.getElementById("notes")

    name.value = event.name;
    date.valueAsDate = Date.parse(event.time);
    notes.value = event.notes;

    Telegram.WebApp.MainButton.setText("Сохранить")

    Telegram.WebApp.MainButton.onClick(async function () {

        if (form.checkValidity() === false) {
            form.reportValidity()
            return
        }

        let songIds = []
        let items = songsElement.getElementsByTagName("span")
        for (let i = 0; i < items.length; i++) {
            let songId = items[i].getAttribute("data-song-id")
            songIds.push(songId)
        }

        let data = JSON.stringify({
            "time": date.valueAsDate,
            "name": name.value,
            "bandId": event.bandId,
            "songIds": songIds,
            "notes": notes.value
        })

        let resp = await fetch(`/web-app/events/${event.id}/edit/confirm?queryId=${Telegram.WebApp.initDataUnsafe.query_id}&messageId=${messageId}&chatId=${chatId}`, {
            method: "POST",
            headers: {'Content-Type': 'application/json'},
            body: data,
        })

        Telegram.WebApp.close()
    })
})()