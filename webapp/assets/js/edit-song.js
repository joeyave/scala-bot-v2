import {Transposer} from 'https://cdn.skypack.dev/chord-transposer';

window.addEventListener('DOMContentLoaded', (e) => {

    Telegram.WebApp.expand()

    let form = document.getElementById('form');
    let name = document.getElementById('name');
    let key = document.getElementById('key');
    let bpm = document.getElementById('bpm');
    let time = document.getElementById('time');
    let tags = document.getElementById("tags")
    let lyrics = document.getElementById("lyrics")
    autosize(lyrics)

    // let lyricsContainer = document.getElementById("lyrics-container")

    // let fontSize = 12;
    // while (lyrics.offsetWidth < lyricsContainer.offsetWidth-10) {
    //     fontSize += 1;
    //     lyrics.style.fontSize = fontSize+"px";
    // }

    form.addEventListener("submit", (e) => e.preventDefault())
    form.addEventListener('input', function (event) {
        Telegram.WebApp.MainButton.show()
    })

    key.onfocus = (e) => {
        console.log("set old val " + e.target.value)
        e.target.setAttribute("data-old-value", e.target.value)
    }

    key.onchange = (e) => {
        const oldKey = e.target.getAttribute("data-old-value")
        console.log("old key " + oldKey)
        e.target.setAttribute("data-old-value", e.target.value)

        let walker = document.createTreeWalker(
            lyrics,
            NodeFilter.SHOW_TEXT,
            null
        )

        while (walker.nextNode()) {
            walker.currentNode.nodeValue = Transposer
                .transpose(walker.currentNode.nodeValue)
                .fromKey(oldKey).toKey(e.target.value).toString()
        }
    }

    Telegram.WebApp.ready()

    if (action === "create") {
        createSong()
    } else {
        editSong()
    }

    function createSong() {
        Telegram.WebApp.MainButton.setText("Создать")
        Telegram.WebApp.MainButton.onClick(function () {

            if (form.checkValidity() === false) {
                form.reportValidity()
                return
            }

            let data = JSON.stringify({
                "name": name.value,
                "key": key.value,
                "bpm": bpm.value,
                "time": time.value,
                "tags": Array.from(tags.selectedOptions)
                    .map(({value}) => value)
                    .filter((s, i) => {
                        if (i !== 0) {
                            return s;
                        }
                    })
            })

            Telegram.WebApp.sendData(data)
        })
    }

    function editSong() {
        lyrics.disabled = true;

        Telegram.WebApp.MainButton.setText("Сохранить")

        Telegram.WebApp.MainButton.onClick(async function () {

                if (form.checkValidity() === false) {
                    form.reportValidity()
                    return
                }

                let data = JSON.stringify({
                    "name": name.value,
                    "key": key.value,
                    "bpm": bpm.value,
                    "time": time.value,
                    "tags": Array.from(tags.selectedOptions)
                        .map(({value}) => value)
                        .filter((s, i) => {
                            if (i !== 0) {
                                return s;
                            }
                        })
                })

                await fetch(`/web-app/songs/${song.id}/edit/confirm?queryId=${Telegram.WebApp.initDataUnsafe.query_id}&messageId=${messageId}&chatId=${chatId}&userId=${userId}`, {
                    method: "POST",
                    headers: {'Content-Type': 'application/json'},
                    body: data,
                })

                Telegram.WebApp.close()
            }
        )

    }
})