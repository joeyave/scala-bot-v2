import {Transposer} from 'https://cdn.skypack.dev/chord-transposer';

window.addEventListener('DOMContentLoaded', (e) => {

    Telegram.WebApp.expand()

    let form = document.getElementById('form')
    let name = document.getElementById('name')
    let key = document.getElementById('key')
    let bpm = document.getElementById('bpm')
    let time = document.getElementById('time')
    let tags = document.getElementById("tags")

    new TomSelect('#tags', {
        plugins: ['remove_button', 'input_autogrow'],
        create: true,
        // createFilter: /^\p{Lu}\p{Ll}+( \p{L}+)*$/gmu,
        placeholder: "Добавить тег"
    });

    Telegram.WebApp.ready()

    Array.from(form.elements).forEach((element) => {
        if (element.tagName === "SELECT" && element.multiple) {
            element.initValue = Array.from(element.selectedOptions).map(({value}) => value)
        } else {
            element.initValue = element.value
        }
    });

    form.addEventListener("submit", (e) => {
        e.preventDefault()
    })
    form.addEventListener('input', (e) => {

        let hide = []
        Array.from(form.elements).forEach((element) => {
            if (element.tagName === "SELECT" && element.multiple) {
                console.log(element.initValue)
                let opts = Array.from(element.selectedOptions).map(({value}) => value)
                console.log(opts)
                hide.push(JSON.stringify(opts) === JSON.stringify(element.initValue))
                console.log(hide);
            } else {
                hide.push(element.initValue === element.value)
            }
        });

        if (!hide.includes(false)) {
            Telegram.WebApp.MainButton.hide()
            console.log("MainButton hidden")
        } else {
            Telegram.WebApp.MainButton.show()
            console.log("MainButton shown")
        }
    })

    if (action === "create") {
        createSong()
    } else {
        editSong()
    }

    function createSong() {
        let lyrics = document.getElementById("lyrics")
        autosize(lyrics)

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
            })

            Telegram.WebApp.sendData(data)
        })
    }

    function editSong() {
        Telegram.WebApp.MainButton.setText("Сохранить")

        let lyricsDiv = document.getElementById("lyrics")
        let initLyricsDiv = lyricsDiv.cloneNode(true)

        key.onchange = (e) => {

            if (e.target.value !== song.pdf.key) {
                document.getElementById("transpose-opts").classList.remove("visually-hidden")
            } else {
                document.getElementById("transpose-opts").classList.add("visually-hidden")
            }

            if (e.target.value === song.pdf.key) {
                lyricsDiv.innerHTML = initLyricsDiv.innerHTML
                return
            }

            let originalKey
            try {
                originalKey = new Transposer(song.pdf.key).getKey()
            } catch (err) {
                console.log(err)
            }

            let newKey
            try {
                newKey = new Transposer(e.target.value).getKey().majorKey
            } catch (err) {
                newKey = e.target.value
            }

            let clone = initLyricsDiv.cloneNode(true)
            let walker = document.createTreeWalker(
                clone,
                NodeFilter.SHOW_TEXT,
                null
            )

            while (walker.nextNode()) {
                if (originalKey) {
                    walker.currentNode.nodeValue = Transposer
                        .transpose(walker.currentNode.nodeValue)
                        .fromKey(originalKey)
                        .toKey(newKey).toString()
                } else {
                    walker.currentNode.nodeValue = Transposer
                        .transpose(walker.currentNode.nodeValue)
                        .toKey(newKey).toString()
                }
            }

            lyricsDiv.innerHTML = clone.innerHTML
        }

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
