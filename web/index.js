app = {
    title: document.getElementById("title"),
    titleText: "SECRET SERVICE",
    form: document.getElementById("form"),
    start(){
        app.animation();
        console.log(document.getElementById("type").value)
        app.type({value:document.getElementById("type").value})
    },
    type(t){
        document.getElementById("download").removeAttribute("disabled")
        app.audio.audioBlob = null
        while (app.form.firstChild) {
            app.form.removeChild(app.form.firstChild);
        }
        switch(t.value){
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
                app.form.innerHTML = `<article style="display:flex;"><button class="control" id="audioplay" onclick="app.audio.play()">&#x23F5; Listen</button><button class="control" id="audiostart" onclick="app.audio.start()">&#x23FA; Record</button><button class="control" onclick="app.audio.cancel()">&#x23CF; Cancel</button></article>`;
            break;
            case "video":
                app.form.innerHTML = `<article style="display:flex;margin:auto;"><video id="videoplayback" style="min-width:12rem;margin:auto;" autoplay></video><div><button class="control" id="videoplay" onclick="app.video.play()">&#x23F5; Listen</button><button class="control" id="videostart" onclick="app.video.start()">&#x23FA; Record</button><button class="control" onclick="app.video.cancel()">&#x23CF; Cancel</button></article></div>`;
            break;
        }
    },
    animation(){
        setInterval(()=>{
            num = Math.floor(Math.random() * (9672 - 9642) ) + 9642;
            char = Math.floor(Math.random() * app.titleText.length);
            app.title.innerHTML = app.titleText.substring(0, char) + `&#${num};` + app.titleText.substring(char + 1);
        },75)
    },
    audio: {
        audioBlobs: [],
        mediaRecorder: null, 
        streamBeingCaptured: null,
        audioPlayer: null,
        audioBlob: null,
        play() {
            this.audioPlayer = document.createElement("audio");
            this.audioPlayer.src = URL.createObjectURL(this.audioBlob)
            this.audioPlayer.play()
        },
        start() {
            if (!(navigator.mediaDevices && navigator.mediaDevices.getUserMedia)) {
                return Promise.reject(new Error('mediaDevices API or getUserMedia method is not supported in this browser.'));
            }
            else {
                return navigator.mediaDevices.getUserMedia({ audio: true })
                    .then(stream => {
                        this.streamBeingCaptured = stream;
                        this.mediaRecorder = new MediaRecorder(stream);
                        this.audioBlobs = [];
                        this.mediaRecorder.addEventListener("dataavailable", event => {
                            this.audioBlobs.push(event.data);
                        });
                        this.mediaRecorder.start();
                        document.getElementById("audiostart").innerHTML = "&#x23F9; Stop"
                        document.getElementById("audiostart").onclick = () => {app.audio.stop()}
                    });
            }
        },
        stop() {
            return new Promise(resolve => {
                let mimeType = this.mediaRecorder.mimeType;
                this.mediaRecorder.addEventListener("stop", () => {
                    this.audioBlob = new Blob(this.audioBlobs, { type: mimeType });
                    resolve(this.audioBlob);
                });
                this.cancel();
            });
        },
        cancel() {
            document.getElementById("audiostart").innerHTML = "&#x23FA; Record"
            document.getElementById("audiostart").onclick = () => {app.audio.start()}
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
    video: {
        videoBlobs: [],
        mediaRecorder: null, 
        streamBeingCaptured: null,
        videoPlayer: null,
        videoBlob: null,
        play() {
            this.videoPlayer = document.getElementById("videoplayback");
            this.videoPlayer.src = URL.createObjectURL(this.videoBlob)
            this.videoPlayer.play()
        },
        start() {
            if (!(navigator.mediaDevices && navigator.mediaDevices.getUserMedia)) {
                return Promise.reject(new Error('mediaDevices API or getUserMedia method is not supported in this browser.'));
            }
            else {
                return navigator.mediaDevices.getUserMedia({ audio: true, video: true })
                    .then(stream => {
                        this.streamBeingCaptured = stream;
                        this.mediaRecorder = new MediaRecorder(stream);
                        this.videoBlobs = [];
                        this.mediaRecorder.addEventListener("dataavailable", event => {
                            this.videoBlobs.push(event.data);
                        });
                        this.mediaRecorder.start();
                        document.getElementById("videostart").innerHTML = "&#x23F9; Stop"
                        document.getElementById("videostart").onclick = () => {app.video.stop()}
                    });
            }
        },
        stop() {
            return new Promise(resolve => {
                let mimeType = this.mediaRecorder.mimeType;
                this.mediaRecorder.addEventListener("stop", () => {
                    this.videoBlob = new Blob(this.videoBlobs, { type: mimeType });
                    resolve(this.videoBlob);
                });
                this.cancel();
            });
        },
        cancel() {
            document.getElementById("videostart").innerHTML = "&#x23FA; Record"
            document.getElementById("videostart").onclick = () => {app.video.start()}
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
        var data = new FormData(document.getElementById("form-parent"))
        data.append("life",parseInt(document.getElementById("life").value))
        data.append("type",document.getElementById("type").value)
        if(app.audio.audioBlob){
            data.append("file",app.audio.audioBlob, `recording.${app.audio.audioBlob.type}`)
        }
        if(app.video.videoBlob){
            data.append("file",app.video.videoBlob, `recording.${app.video.videoBlob.type}`)
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
                if(json.state){
                    try{
                        navigator.clipboard.writeText(json.url)
                    } catch {
                        alert(`We cannot access your clipboard, the url is: ${json.url}`)
                    }
                    document.getElementById("submit").setAttribute("aria-invalid","false")
                    document.getElementById("submit").innerText = "Success: Link copied to clipboard!"
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