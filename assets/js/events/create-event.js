(function () {
    Telegram.WebApp.ready()

    let inputs = [
        document.getElementById("date"),
        document.getElementById("name"),
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

    document.getElementById("date").valueAsDate = new Date();

    Telegram.WebApp.MainButton.setText("Создать")

    Telegram.WebApp.MainButton.onClick(function () {
        Telegram.WebApp.sendData(JSON.stringify({
            "method": "createEvent",
            "event": {
                "name": document.getElementById("name").value,
                "date": document.getElementById("date").value
            }
        }))
    })
})()