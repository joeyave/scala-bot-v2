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

    form.addEventListener("submit", (e) => e.preventDefault())
    form.addEventListener('input', function (event) {
        Telegram.WebApp.MainButton.show()
    })

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
            Telegram.WebApp.MainButton.showProgress()

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
        let lyricsContainer = document.getElementById("lyrics-container")
        let lyricsDiv = document.getElementById("lyrics")

        // console.log (lyricsContainer.offsetWidth - lyricsDiv.offsetWidth)
        //  if ((lyricsContainer.offsetWidth - lyricsDiv.offsetWidth) < 0 || (lyricsContainer.offsetWidth - lyricsDiv.offsetWidth) > 0) {
        //      let fontSize = 24
        //      for (let i = 0; i < 20; i++) {
        //          console.log(lyricsContainer.offsetWidth - lyricsDiv.offsetWidth)
        //          if ((lyricsContainer.offsetWidth - lyricsDiv.offsetWidth) < 0) {
        //              break
        //          }
        //          fontSize -= 1
        //          lyricsDiv.style.fontSize = fontSize + "px !important"
        //      }
        //  }

        // lyricsDiv.style.fontSize = "10px";
        // let fontSize = 10
        // console.log("lyricsDiv.offsetWidth " + lyricsDiv.offsetWidth)
        // console.log(" lyricsContainer.offsetWidth " + lyricsContainer.offsetWidth)
        // while (lyricsDiv.offsetWidth < lyricsContainer.offsetWidth) {
        //     console.log("lyricsDiv.offsetWidth " + lyricsDiv.offsetWidth)
        //     console.log(" lyricsContainer.offsetWidth " + lyricsContainer.offsetWidth)
        //     lyrics.style.fontSize = fontSize + "px";
        //     fontSize += 1;
        // }
        // while (lyricsDiv.offsetWidth >= lyricsContainer.offsetWidth) {
        //     console.log("lyricsDiv.offsetWidth " + lyricsDiv.offsetWidth)
        //     console.log(" lyricsContainer.offsetWidth " + lyricsContainer.offsetWidth)
        //     fontSize -= 1;
        //     lyrics.style.fontSize = fontSize + "px";
        // }


        key.onfocus = (e) => {
            e.target.setAttribute("data-old-value", e.target.value)
        }

        key.onchange = (e) => {

            console.log(song.pdf.key);
            console.log(e.target.value);
            if (e.target.value !== song.pdf.key) {
                document.getElementById("transpose-opts").classList.remove("visually-hidden")
            } else {
                document.getElementById("transpose-opts").classList.add("visually-hidden")
            }

            const oldKey = e.target.getAttribute("data-old-value")
            e.target.setAttribute("data-old-value", e.target.value)

            let walker = document.createTreeWalker(
                lyricsDiv,
                NodeFilter.SHOW_TEXT,
                null
            )

            while (walker.nextNode()) {
                if (oldKey !== "?") {
                    walker.currentNode.nodeValue = Transposer
                        .transpose(walker.currentNode.nodeValue)
                        .fromKey(oldKey).toKey(e.target.value).toString()
                } else {
                    walker.currentNode.nodeValue = Transposer
                        .transpose(walker.currentNode.nodeValue)
                        .toKey(e.target.value).toString()
                }
            }

            // lyricsDiv.innerText = Transposer.transpose(lyricsDiv.innerText).fromKey(oldKey).toKey(e.target.value).toString()
        }

        Telegram.WebApp.MainButton.setText("Сохранить")

        Telegram.WebApp.MainButton.onClick(async function () {

                let transposeSection = document.getElementById("transpose-section")

                if (form.checkValidity() === false) {
                    form.reportValidity()
                    return
                }

                Telegram.WebApp.MainButton.showProgress()

                let data = JSON.stringify({
                    "name": name.value,
                    "key": key.value,
                    "transposeSection": transposeSection.value,
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