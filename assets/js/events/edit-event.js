(function () {
    Telegram.WebApp.ready()

    let inputs = [
        document.getElementById("date"),
        document.getElementById("name"),
        // document.getElementById("notes"),
    ]

    Telegram.WebApp.MainButton.hide()

    for (let i = 0; i < inputs.length; i++) {
        inputs[i].addEventListener('input', () => {
            let values = []
            inputs.forEach(v => values.push(v.value))
            if (values.includes('')) {
                Telegram.WebApp.MainButton.hide()
            } else {
                Telegram.WebApp.MainButton.show()
            }
        })
    }

    document.getElementById("id").value = event.id;
    document.getElementById("name").value = event.name;
    document.getElementById("date").valueAsDate = new Date(event.time); // todo
    document.getElementById("notes").value = event.notes;

    Telegram.WebApp.MainButton.setText("Сохранить")

    Telegram.WebApp.MainButton.onClick(async function () {
        let data = JSON.stringify({
            "event": {
                "name": document.getElementById("name").value,
                "date": document.getElementById("date").value,
                "notes": document.getElementById("notes").value
            },
        })

        let resp = await fetch(`/web-app/events/${event.id}/edit/confirm?queryId=${Telegram.WebApp.initDataUnsafe.query_id}`, {
            method: "POST",
            headers: {'Content-Type': 'application/json'},
            body: data,
        })

        Telegram.WebApp.close()
    })
})()