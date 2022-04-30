window.addEventListener('DOMContentLoaded', (e) => {
    Telegram.WebApp.expand()

    autosize()

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

            let data = JSON.stringify({
                "name": name.value,
                "key": key.value,
                "bpm": bpm.value,
                "time": time.value,
                "tags": Array.from(tags.selectedOptions).map(({value}, i) => {
                    if (i === 0) return
                    return value
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
                "tags": Array.from(tags.selectedOptions).map(({value}, i) => {
                    if (i === 0) {
                        return
                    }
                    return value
                })
            })

            await fetch(`/web-app/songs/${song.id}/edit/confirm?queryId=${Telegram.WebApp.initDataUnsafe.query_id}&messageId=${messageId}&chatId=${chatId}&userId=${userId}`, {
                method: "POST",
                headers: {'Content-Type': 'application/json'},
                body: data,
            })

            Telegram.WebApp.close()
        })

    }
})