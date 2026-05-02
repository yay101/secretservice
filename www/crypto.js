function encode(message) {
    let enc = new TextEncoder();
    return enc.encode(message);
}

async function encrypt(data, iv, key) {
    return window.crypto.subtle.encrypt(
        { name: "AES-GCM", iv: iv },
        key,
        data,
    );
}

async function decrypt(data, iv, key) {
    return window.crypto.subtle.decrypt(
        { name: "AES-GCM", iv: iv },
        key,
        data,
    );
}