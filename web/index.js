app = {
    title: document.getElementById("title"),
    titleText: "SECRET SERVICE",
    form: document.getElementById("form"),
    lastlink: null,
    submitbtn: document.getElementById("submit"),
    iphonecheck(){
        return ['iPad Simulator','iPhone Simulator','iPod Simulator','iPad','iPhone','iPod'].includes(navigator.platform) || (navigator.userAgent.includes("Mac") && "ontouchend" in document)
    },
    newlink(url){
        document.getElementById("links").innerHTML += `<linkitem><span>${url}</span><button type="button" onclick="app.oldclipboard(this)"><img style="height:1.5rem;" src="/img/clipboard2.svg" class="icon"></button></linkitem>`
    },
    toclipboard(){
        if(app.lastlink){
            app.newlink(app.lastlink)
            navigator.clipboard.writeText(app.lastlink)
            .then(result => {
                document.getElementById("cbimg").src="/img/clipboard2-check.svg"
                app.submitbtn.innerText = "Success: Link copied to clipboard!"
            })
            .catch(err => {app.submitbtn.innerText = "Grab the link below!";console.log(err)})
            setTimeout(()=>{
                document.getElementById("cbimg").src="/img/clipboard2.svg"
                app.submitbtn.innerText = "Get Another Link"
            },3000)
        }
    },
    oldclipboard(e){
        navigator.clipboard.writeText(e.parentNode.innerText)
        e.firstChild.src="/img/clipboard2-check.svg"
        setTimeout(()=>{
            e.firstChild.src="/img/clipboard2.svg"
        },3000)
    },
    start(){
        app.animation();
        app.type({value:document.getElementById("type").value});
        console.log(app.iphonecheck())
        if(app.iphonecheck == true){
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
                app.form.innerHTML += `<textarea style="min-height: 3rem;" name="secret" placeholder="Tell me your secret"></textarea>`
            break;
            case "binary":
                app.form.innerHTML = `<input type="file" id="file" name="file">`;
                document.getElementById("download").setAttribute("disabled","true")
            break;
            case "audio":
                app.form.style = "display:flex;"
                app.form.innerHTML = `<audio id="audio"></audio><button class="control" id="playback" onclick="app.media.play()"><img class="icon" src="/img/play.svg">Play</button><button class="control" id="record" onclick="app.media.start()"><img class="icon" src="/img/record2.svg">Record</button><button class="control" onclick="app.media.cancel()"><img class="icon" src="/img/eject.svg">Cancel</button>`;
                app.media.mediaPlayer = document.getElementById("audio");
            break;
            case "video":
                app.form.style = "display:flex;"
                app.form.innerHTML = `<video id="video" style="min-width:12rem;margin:auto;width:100%;" autoplay controls></video><div id="controls" style="display:flex;"><button class="control" id="playback" onclick="app.media.play()"><img class="icon" src="/img/play.svg">Play</button><button class="control" id="record" onclick="app.media.start()"><img class="icon" src="/img/record2.svg">Record</button><button class="control" onclick="app.media.cancel()"><img class="icon" src="/img/eject.svg">Cancel</button></div>`;
                app.media.mediaPlayer = document.getElementById("video");
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
            this.mediaPlayer.src = URL.createObjectURL(this.mediaBlob)
            this.mediaPlayer.play()
        },
        start() {
            if (!(navigator.mediaDevices && navigator.mediaDevices.getUserMedia)) {
                return Promise.reject(new Error('mediaDevices API or getUserMedia method is not supported in this browser.'));
            }
            else {
                var options
                if(document.getElementById("type").value == "audio"){
                    options = { audio: true }
                } else {
                    options = { audio: true, video: true}
                }
                return navigator.mediaDevices.getUserMedia(options)
                .then(stream => {
                    this.streamBeingCaptured = stream;
                    this.mediaRecorder = new MediaRecorder(stream);
                    this.mediaBlobs = [];
                    this.mediaRecorder.addEventListener("dataavailable", event => {
                        this.mediaBlobs.push(event.data);
                    });
                    this.mediaRecorder.start();
                    document.getElementById("record").onclick = ()=>{app.media.stop()}
                    document.getElementById("record").innerHTML = `<img class="icon" src="/img/stop.svg">Stop`
                });
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
        app.submitbtn.setAttribute("aria-busy","true")
        data = new FormData(document.getElementById("form-parent"));
        data.append("type",document.getElementById("type").value)
        data.append("life",Number(document.getElementById("life").value))
        grecaptcha.execute(document.querySelector("html").dataset.recaptcha)
        .then(token => {
            fetch("/service/",{
                method:"POST",
                headers: {
                    "X-Captcha-Token": token,
                },
                body: data,
            })
            .then(response => response.json())
            .then(json => {
                app.submitbtn.setAttribute("aria-busy","false")
                if(json.state){
                    app.lastlink = json.url;
                    app.toclipboard()
                } else {
                    app.submitbtn.innerText = "Error: Something went wrong."
                    app.submitbtn.setAttribute("aria-invalid","true")
                }
                setTimeout(()=>{app.submitbtn.innerText = "Get Link";app.submitbtn.removeAttribute("aria-invalid")},3000)
                document.getElementById("form-parent").reset()
                console.log(json)
            })
        })
    },
}
app.start()