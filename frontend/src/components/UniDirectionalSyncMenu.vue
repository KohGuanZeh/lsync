<script lang="ts" setup>
import { reactive } from 'vue'
import { SelectDirectory } from '../../wailsjs/go/main/App'

type DirType = "src" | "dst";

const data = reactive({
    srcDir: "",
    dstDir: ""
})

function selectFolder(dirType: DirType) {
    const title = dirType == "src" ? "Source Directory" : "Destination Directory";
    SelectDirectory(title).then(res => {
        if (dirType == "src") {
            data.srcDir = res;
            return;
        }
        data.dstDir = res;
    })
}

</script>

<template>
    <main>
        <section class="lsync-controls">
            <div id="src-input" class="dir-select-container">
                <label for="src-dir">Source</label>
                <div class="dir-select">
                    <input id="src-dir" name="src-dir" v-model="data.srcDir" placeholder="Select Source Folder"
                        autocomplete="off" type="text" />
                    <button @click="selectFolder('src')">Select Folder</button>
                </div>
            </div>
            <div id="dst-input" class="dir-select-container">
                <label for="dst-dir">Destination</label>
                <div class="dir-select">
                    <input id="dst-dir" name="dst-dir" v-model="data.dstDir" placeholder="Select Destination Folder"
                        autocomplete="off" type="text" />
                    <button @click="selectFolder('dst')">Select Folder</button>
                </div>
            </div>
            <button class="sync-button">
                Preview
            </button>
            <button class="sync-button">
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

.sync-button {
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
