(function () {
    Telegram.WebApp.expand()

    const dragArea = document.querySelector(".songs-wrapper");

    new Sortable(dragArea, {
        handle: ".fa-bars",  // Drag handle selector within list items
        draggable: ".item",  // Specifies which items inside the element should be draggable
        animation: 0,
        onUpdate: function (/**Event*/evt) {
            Telegram.WebApp.MainButton.show()
        },
    });

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
        let items = dragArea.getElementsByTagName("span")
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