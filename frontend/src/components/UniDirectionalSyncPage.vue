<script lang="ts" setup>
import { reactive, toRef } from 'vue'
import { PreviewSync, PreviewSyncAsync, SyncWithPreview } from '../../wailsjs/go/backend/App'
import SyncPreviewTree from './SyncPreviewTree.vue';
import { lsync } from '../../wailsjs/go/models';
import DirectorySelect, { DirType } from './DirectorySelect.vue'

interface SyncInfo {
    srcDir: string,
    dstDir: string,
    dirPreview?: lsync.SyncPreview
    ignoreMissingInSrc: boolean
}

const syncInfo: SyncInfo = reactive({
    srcDir: "",
    dstDir: "",
    dirPreview: undefined,
    ignoreMissingInSrc: false
});

function updateDirVal(dirType: DirType, newDirVal: string) {
    const refDir = toRef(syncInfo, dirType == DirType.Src ? "srcDir" : "dstDir");
    if (refDir.value != newDirVal) {
        refDir.value = newDirVal;
        syncInfo.dirPreview = undefined;
    }
}

function previewSync() {
    if (!syncInfo.srcDir || !syncInfo.dstDir) {
        return;
    }
    PreviewSync(syncInfo.srcDir, syncInfo.dstDir).then(res => {
        syncInfo.dirPreview = res;
    }, err => {
        console.log(err);
        syncInfo.dirPreview = undefined;
    });
}

function previewSyncAsync() {
    if (!syncInfo.srcDir || !syncInfo.dstDir) {
        return;
    }
    PreviewSyncAsync(syncInfo.srcDir, syncInfo.dstDir).then(res => {
        syncInfo.dirPreview = res;
    }, err => {
        console.log(err);
        syncInfo.dirPreview = undefined;
    });
}

function syncFolders() {
    if (!syncInfo.dirPreview) {
        return;
    }
    SyncWithPreview(syncInfo.srcDir, syncInfo.dstDir, syncInfo.dirPreview, syncInfo.ignoreMissingInSrc).then(res => {
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
            <h1>Unidirectional Sync</h1>
            <section class="dir-select">
                <DirectorySelect :dirType="DirType.Src" :dirVal="syncInfo.srcDir" @update-dir-value="updateDirVal">
                </DirectorySelect>
                <DirectorySelect :dirType="DirType.Dst" :dirVal="syncInfo.dstDir" @update-dir-value="updateDirVal">
                </DirectorySelect>
            </section>
            <section class="additional-sync-options">
                <div class="additional-sync-options-header">Additional Sync Options:</div>
                <div class="checkbox-option">
                    <label class="checkbox-label">
                        <input type="checkbox" name="ignoreMissingInSrc" v-model="syncInfo.ignoreMissingInSrc">
                        Ignore directories and files missing in source
                    </label>
                </div>
            </section>
            <section class="sync-controls">
                <button class="btn" @click="previewSync">Preview</button>
                <button class="btn" @click="previewSyncAsync">Preview Async</button>
            </section>
        </section>
        <section class="dir-sync-preview" v-if="syncInfo.dirPreview">
            <SyncPreviewTree :dir-path="syncInfo.dstDir" :dirName="syncInfo.dstDir" :dirPreview="syncInfo.dirPreview"
                :ignore-deleted="syncInfo.ignoreMissingInSrc">
            </SyncPreviewTree>
        </section>
    </main>
</template>

<style scoped>
h1 {
    text-align: center;
    margin: 0;
    margin-bottom: 1rem;
}

.uni-dir-sync {
    width: 100%;
    display: flex;
    flex-direction: column;
}

.dir-select {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 2rem;
    padding-bottom: 1rem;
}

.additional-sync-options {
    padding-bottom: 1rem;
}

.additional-sync-options-header {
    padding-bottom: 0.5rem;
}

.checkbox-option {
    display: flex;
    gap: 0.5rem;
}

.checkbox-label {
    cursor: pointer;
}

.sync-controls {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 2rem;
    padding-bottom: 1rem;
}

.btn {
    flex: 1;
}

.dir-sync-preview {
    max-height: 55vh;
    min-height: 420px;
    background-color: #505050;
    border-radius: 0.25rem;
    padding: 1rem;
    display: flex;
    overflow: auto;
}
</style>
