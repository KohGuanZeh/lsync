<script lang="ts" setup>
import { ref } from 'vue'
import { PreviewSync, SelectDirectory } from "../../wailsjs/go/backend/App"

enum DirType {
    Src = "src",
    Dst = "dst"
}

const srcDir = ref("");
const dstDir = ref("");

function selectFolder(dirType: DirType) {
    let title = "Select Source Directory";
    let dirRef = srcDir;
    if (dirType != DirType.Src) {
        title = "Select Destination Directory";
        dirRef = dstDir;
    }
    SelectDirectory(title).then(res => {
        if (res.length != 0) {
            dirRef.value = res;
        }
    }, err => {
        console.log(err);
    });
}

function previewSync() {
    PreviewSync(srcDir.value, dstDir.value).then(res => {
        console.log(res);
    }, err => {
        console.log(err);
    })
}

</script>

<template>
    <main>
        <section class="lsync-controls">
            <div id="src-input" class="dir-select-container">
                <label for="src-dir">Source</label>
                <div class="dir-select">
                    <input id="src-dir" name="src-dir" v-model="srcDir" placeholder="Select Source Folder"
                        autocomplete="off" type="text" />
                    <button @click="selectFolder(DirType.Src)">Select Folder</button>
                </div>
            </div>
            <div id="dst-input" class="dir-select-container">
                <label for="dst-dir">Destination</label>
                <div class="dir-select">
                    <input id="dst-dir" name="dst-dir" v-model="dstDir" placeholder="Select Destination Folder"
                        autocomplete="off" type="text" />
                    <button @click="selectFolder(DirType.Dst)">Select Folder</button>
                </div>
            </div>
            <button class="btn" @click="previewSync">
                Preview
            </button>
            <button class="btn">
                Sync
            </button>
        </section>
    </main>
</template>

<style scoped>
main {
    width: 100%;
    display: flex;
}

.btn {
    margin-top: auto;
    max-height: fit-content;
}

.lsync-controls {
    margin: 0 auto;
    display: flex;
    gap: 3rem;
}

.dir-select-container {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
}

.dir-select {
    display: flex;
    gap: 0.25rem;
}
</style>
