import InstantSearch from "../components/instant_search/InstantSearch.js";

window.addEventListener('DOMContentLoaded', (e) => {
    Telegram.WebApp.expand()

    const songsElement = document.querySelector(".sortable-list");
    const searchElement = document.querySelector("#search");
    const overlayElement = document.querySelector(".overlay")
    const addSongButton = document.querySelector("#add-song-button")
    let form = document.getElementById('event-form');

    new Sortable(songsElement, {
        delay: 150,
        delayOnTouchOnly: true,
        animation: 100,
        onUpdate: function (/**Event*/evt) {
            Telegram.WebApp.MainButton.show()
        },
    });

    new InstantSearch(searchElement, {
        searchUrl: new URL(`/api/drive-files/search?driveFolderId=${event.band.driveFolderId}`, window.location.origin),
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
                    // overlayElement.classList.add("overlay--hidden")

                    let resp = await fetch(`/api/songs/find-by-drive-file-id?driveFileId=${result.id}`, {
                        method: "get",
                        headers: {'Content-Type': 'application/json'},
                    })

                    let data = await resp.json()

                    Telegram.WebApp.MainButton.show()
                    // overlayElement.style.display = "none";

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

    Telegram.WebApp.ready()

    let name = document.getElementById("name")
    let date = document.getElementById("date")
    let notes = document.getElementById("notes")

    if (action === "create") {
        createEvent()
    } else {
        editEvent(event)
    }

    // Adding listeners.
    // overlayElement.onclick = (e) => {
    //     if (e.target !== overlayElement) return
    //     overlayElement.classList.add("overlay--hidden")
    // }

    // addSongButton.addEventListener("click", () => {
    //     console.log("click")
    //     overlayElement.classList.remove("overlay--hidden")
    //     document.getElementById("song-search-input").focus()
    // })

    document.addEventListener("click", (e) => {
        if (e.target.id === "delete-song-icon") {
            e.target.parentElement.remove()
            Telegram.WebApp.MainButton.show()
        }
    })

    form.addEventListener('input', function (event) {
        Telegram.WebApp.MainButton.show()
    })


    function editEvent(event) {
        Telegram.WebApp.MainButton.setText("Сохранить")

        name.value = event.name;
        date.value = new Date(event.time).toISOString().substring(0, 10);
        notes.value = event.notes;

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

            await fetch(`/web-app/events/${event.id}/edit/confirm?queryId=${Telegram.WebApp.initDataUnsafe.query_id}&messageId=${messageId}&chatId=${chatId}`, {
                method: "POST",
                headers: {'Content-Type': 'application/json'},
                body: data,
            })

            Telegram.WebApp.close()
        })
    }

    function createEvent() {
        Telegram.WebApp.MainButton.setText("Создать")

        date.value = new Date().toISOString().substring(0, 10);

        Telegram.WebApp.MainButton.onClick(function () {

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
                // "bandId": event.bandId,
                "songIds": songIds,
                "notes": notes.value
            })

            Telegram.WebApp.sendData(data)
        })
    }
});