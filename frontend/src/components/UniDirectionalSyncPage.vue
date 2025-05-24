<script lang="ts" setup>
import { ref } from 'vue'
import { PreviewSync, SelectDirectory } from '../../wailsjs/go/backend/App'
import SyncPreviewTree from './SyncPreviewTree.vue';
import { dirsyncmap } from '../../wailsjs/go/models';
import DirectorySelect, { DirType } from './DirectorySelect.vue'

const srcDir = ref("");
const dstDir = ref("");
const dirPreview = ref<dirsyncmap.DirSyncStruct | undefined>(undefined);

function updateDirVal(dirType: DirType, newDirVal: string) {
    const refDir = dirType == DirType.Src ? srcDir : dstDir;
    refDir.value = newDirVal;
    dirPreview.value = undefined;
}

function previewSync() {
    PreviewSync(srcDir.value, dstDir.value).then(res => {
        dirPreview.value = res;
    }, err => {
        console.log(err);
        dirPreview.value = undefined;
    })
}
</script>

<template>
    <main>
        <section class="uni-dir-sync">
            <section class="dir-select">
                <DirectorySelect :dirType="DirType.Src" :dirVal="srcDir" @update-dir-value="updateDirVal">
                </DirectorySelect>
                <DirectorySelect :dirType="DirType.Dst" :dirVal="dstDir" @update-dir-value="updateDirVal">
                </DirectorySelect>
            </section>
            <section class="sync-controls">
                <button class="sync-control-btn" @click="previewSync">Preview</button>
                <button class="sync-control-btn">Sync</button>
            </section>
        </section>
        <section class="dir-sync-preview" v-if="dirPreview">
            <SyncPreviewTree :dirPreview="dirPreview" :dirName="dstDir"></SyncPreviewTree>
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

.sync-control-btn {
    flex: 1;
    text-align: center;
}

.dir-sync-preview {
    background-color: #505050;
    border-radius: 0.25rem;
    padding: 1rem;
    display: flex;
    overflow: auto;
}
</style>
