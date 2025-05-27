<script lang="ts" setup>
import { reactive, ref, toRef } from 'vue'
import { PreviewSync, SyncWithPreview } from '../../wailsjs/go/backend/App'
import SyncPreviewTree from './SyncPreviewTree.vue';
import { sync } from '../../wailsjs/go/models';
import DirectorySelect, { DirType } from './DirectorySelect.vue'

interface SyncData {
    srcDir: string,
    dstDir: string,
    dirPreview?: sync.SyncPreview
}

const syncSettings: SyncData = reactive({
    srcDir: "",
    dstDir: "",
    dirPreview: undefined
});

function updateDirVal(dirType: DirType, newDirVal: string) {
    const refDir = toRef(syncSettings, dirType == DirType.Src ? "srcDir" : "dstDir");
    if (refDir.value != newDirVal) {
        refDir.value = newDirVal;
        syncSettings.dirPreview = undefined;
    }
}

function previewSync() {
    if (!syncSettings.srcDir || !syncSettings.dstDir) {
        return;
    }
    PreviewSync(syncSettings.srcDir, syncSettings.dstDir).then(res => {
        syncSettings.dirPreview = res;
    }, err => {
        console.log(err);
        syncSettings.dirPreview = undefined;
    });
}

function syncFolders() {
    if (!syncSettings.dirPreview) {
        return;
    }
    SyncWithPreview(syncSettings.srcDir, syncSettings.dstDir, syncSettings.dirPreview).then(res => {
        console.log("Success");
    }, err => {
        console.log(err);
    });
    previewSync();
}
</script>

<template>
    <main>
        <section class="uni-dir-sync">
            <section class="dir-select">
                <DirectorySelect :dirType="DirType.Src" :dirVal="syncSettings.srcDir" @update-dir-value="updateDirVal">
                </DirectorySelect>
                <DirectorySelect :dirType="DirType.Dst" :dirVal="syncSettings.dstDir" @update-dir-value="updateDirVal">
                </DirectorySelect>
            </section>
            <section class="sync-controls">
                <button class="btn" @click="previewSync">Preview</button>
                <button class="btn" @click="syncFolders">Sync</button>
            </section>
        </section>
        <section class="dir-sync-preview" v-if="syncSettings.dirPreview">
            <SyncPreviewTree :dir-path="syncSettings.dstDir" :dirName="syncSettings.dstDir"
                :dirPreview="syncSettings.dirPreview"></SyncPreviewTree>
        </section>
    </main>
</template>

<style scoped>
.uni-dir-sync {
    width: 100%;
    display: flex;
    flex-direction: column;
}

.dir-select {
    width: 100%;
    margin: 0 auto;
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 2rem;
    padding-bottom: 1.5rem;
}

.sync-controls {
    width: 100%;
    margin: 0 auto;
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 2rem;
    padding-bottom: 1.5rem;
}

.btn {
    flex: 1;
}

.dir-sync-preview {
    max-height: 70vh;
    min-height: 512px;
    background-color: #505050;
    border-radius: 0.25rem;
    padding: 1rem;
    display: flex;
    overflow: auto;
}
</style>
