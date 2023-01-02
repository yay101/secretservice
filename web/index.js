app = {
    title: document.getElementById("title"),
    titleText: "SECRET SERVICE",
    form: document.getElementById("form"),
    lastlink: null,
    iphonecheck(){
        return ['iPad Simulator','iPhone Simulator','iPod Simulator','iPad','iPhone','iPod'].includes(navigator.platform) || (navigator.userAgent.includes("Mac") && "ontouchend" in document)
    },
    toclipboard(){
        if(app.lastlink){
            try{
                navigator.clipboard.writeText(app.lastlink)
                document.getElementById("cbimg").src="/img/clipboard2-check.svg"
                document.getElementById("submit").innerText = "Success: Link copied to clipboard!"
                setTimeout(()=>{
                    document.getElementById("cbimg").src="/img/clipboard2.svg"
                    document.getElementById("submit").innerText = "Get Another Link"
                },3000)
            } catch {
                document.getElementById("submit").innerText = "Warning: Could not copy to clipboard! Click >"
            }
        }
    },
    start(){
        app.animation();
        app.type({value:document.getElementById("type").value});
        if(app.iphonecheck){
            for(opt of document.getElementsByTagName("option")){
                if(["audio","video"].includes(opt.value)){
                    opt.setAttribute("disabled","true");
                }
            }
        }
    },
    type(type){
        document.getElementById("download").removeAttribute("disabled")
        app.media.mediaBlob = null
        while (app.form.firstChild) {
            app.form.removeChild(app.form.firstChild);
        }
        switch(type.value){
            case "string":
                textinput = document.createElement("input");
                textinput.type = "text";
                textinput.name = "secret"
                textinput.placeholder = "Tell me your secret"
                app.form.appendChild(textinput);
            break;
            case "binary":
                app.form.innerHTML = `<input type="file" id="file" name="file">`;
                document.getElementById("download").setAttribute("disabled","true")
            break;
            case "audio":
                app.form.style = "display:flex;"
                app.form.innerHTML = `<button class="control" id="playback" onclick="app.media.play()"><img class="icon" src="/img/play.svg">Play</button><button class="control" id="record" onclick="app.media.start()"><img class="icon" src="/img/record2.svg">Record</button><button class="control" onclick="app.media.cancel()"><img class="icon" src="/img/eject.svg">Cancel</button>`;
            break;
            case "video":
                app.form.style = "display:flex;margin:auto;"
                app.form.innerHTML = `<video id="videomonitor" style="min-width:12rem;margin:auto;" autoplay></video><div><button class="control" id="playback" onclick="app.media.play()"><img class="icon" src="/img/play.svg">Play</button><button class="control" id="record" onclick="app.media.start()"><img class="icon" src="/img/record2.svg">Record</button><button class="control" onclick="app.media.cancel()"><img class="icon" src="/img/eject.svg">Cancel</button></div>`;
            break;
        }
    },
    animation(){
        int = setInterval(()=>{
            num = Math.floor(Math.random() * (9672 - 9642) ) + 9642;
            char = Math.floor(Math.random() * app.titleText.length);
            app.title.innerHTML = app.titleText.substring(0, char) + `&#${num};` + app.titleText.substring(char + 1);
        },75)
        setTimeout(()=>{
            clearInterval(int)
        },3000)
    },
    media: {
        mediaBlobs: [],
        mediaRecorder: null, 
        streamBeingCaptured: null,
        mediaPlayer: null,
        mediaBlob: null,
        play() {
            switch(document.getElementById("type").value){
                case "audio":
                    this.mediaPlayer = document.createElement("audio");
                    this.mediaPlayer.src = URL.createObjectURL(this.mediaBlob)
                    this.mediaPlayer.play()
                break;
                case "video":
                    this.mediaPlayer = document.getElementById("videomonitor");
                    this.mediaPlayer.src = URL.createObjectURL(this.mediaBlob)
                    this.mediaPlayer.play()
                break;
            }
        },
        start() {
            if (!(navigator.mediaDevices && navigator.mediaDevices.getUserMedia)) {
                return Promise.reject(new Error('mediaDevices API or getUserMedia method is not supported in this browser.'));
            }
            else {
                document.getElementById("record").onclick = ()=>{app.media.stop()}
                document.getElementById("record").innerHTML = `<img class="icon" src="/img/stop.svg">Stop`
                switch(document.getElementById("type").value){
                    case "audio":
                        return navigator.mediaDevices.getUserMedia({ audio: true })
                        .then(stream => {
                            this.streamBeingCaptured = stream;
                            this.mediaRecorder = new MediaRecorder(stream);
                            this.mediaBlobs = [];
                            this.mediaRecorder.addEventListener("dataavailable", event => {
                                this.mediaBlobs.push(event.data);
                            });
                            this.mediaRecorder.start();
                        });
                    case "video":
                        return navigator.mediaDevices.getUserMedia({ audio: true, video: true})
                        .then(stream => {
                            this.streamBeingCaptured = stream;
                            this.mediaRecorder = new MediaRecorder(stream, options);
                            this.mediaBlobs = [];
                            this.mediaRecorder.addEventListener("dataavailable", event => {
                                this.mediaBlobs.push(event.data);
                            });
                            this.mediaRecorder.start();
                        });
                }
            }
        },
        stop() {
            return new Promise(resolve => {
                let mimeType = this.mediaRecorder.mimeType;
                this.mediaRecorder.addEventListener("stop", () => {
                    this.mediaBlob = new Blob(this.mediaBlobs, { type: mimeType });
                    resolve(this.mediaBlob);
                });
                this.cancel();
                document.getElementById("record").innerHTML = `<img class="icon" src="/img/record2.svg">Record`
                document.getElementById("record").onclick = ()=>{app.media.start()}
            });
        },
        cancel() {
            this.mediaRecorder.stop();
            this.stopStream();
            this.resetRecordingProperties();
        },
        stopStream() {
            this.streamBeingCaptured.getTracks()
                .forEach(track => track.stop());
        },
        resetRecordingProperties() {
            this.mediaRecorder = null;
            this.streamBeingCaptured = null;
        }
    },
    send(){
        document.getElementById("submit").setAttribute("aria-busy","true")
        var data = new FormData(document.getElementById("form-parent"))
        data.append("life",parseInt(document.getElementById("life").value))
        data.append("type",document.getElementById("type").value)
        if(app.media.mediaBlob){
            data.append("file",app.media.mediaBlob, `recording.${app.media.mediaBlob.type}`)
        }
        grecaptcha.execute(document.querySelector("html").dataset.recaptcha)
        .then(token => {
            data.append("token",token)
            fetch("/service",{
                method:"POST",
                body: data,
            })
            .then(response => response.json())
            .then(json => {
                document.getElementById("submit").setAttribute("aria-busy","false")
                if(json.state){
                    app.lastlink = json.url;
                    app.toclipboard()
                } else {
                    document.getElementById("submit").innerText = "Error: Something went wrong."
                    document.getElementById("submit").setAttribute("aria-invalid","true")
                }
                setTimeout(()=>{document.getElementById("submit").innerText = "Get Link";document.getElementById("submit").removeAttribute("aria-invalid")},3000)
                document.getElementById("form-parent").reset()
                console.log(json)
            })
        })
    },
}
app.start()