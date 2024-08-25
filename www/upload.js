onmessage = (msg) => {
  url = msg.data.url;
  file = msg.data.file;
  key = msg.data.key;
  iv = msg.data.iv;
  chunkSize = Number(msg.data.cs);
  async function encrypt(data) {
    return self.crypto.subtle.encrypt({ name: "AES-GCM", iv: iv }, key, data);
  }
  async function onLoadHandler(event) {
    let result = event.target.result;
    if (event.target.error == null) {
      await fetch(url, {
        method: "POST",
        headers: {
          "Content-Type": "application/octet-stream",
          "Content-Range": chunk + "/" + (chunks - 1),
        },
        body: await encrypt(result, iv, key),
      }).then((res) => {
        if (res.ok) {
          offset += event.target.result.byteLength;
          chunk++;
          retry = 0;
        } else {
          if (retry >= 3) {
            postMessage(-1);
            return;
          }
          retry++;
        }
        postMessage(offset);
        if (offset < fileSize) {
          readBlock(offset, chunkSize, file);
        }
      });
    } else {
      postMessage(-1);
      return;
    }
  }
  async function readBlock(offset, length, file) {
    let fileReader = new FileReader();
    let blob = file.slice(offset, length + offset);
    fileReader.onload = await onLoadHandler;
    fileReader.readAsArrayBuffer(blob);
  }
  var fileSize = file.size;
  var chunks = Math.ceil(fileSize / chunkSize);
  var offset = 0;
  var chunk = 0;
  var retry = 0;
  readBlock(offset, chunkSize, file);
};
