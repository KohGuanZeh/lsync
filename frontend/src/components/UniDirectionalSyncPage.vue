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
        <section class="lsync-controls">
            <DirectorySelect :dirType="DirType.Src" :dirVal="srcDir" @update-dir-value="updateDirVal"></DirectorySelect>
            <DirectorySelect :dirType="DirType.Dst" :dirVal="dstDir" @update-dir-value="updateDirVal"></DirectorySelect>
            <button class="btn" @click="previewSync">
                Preview
            </button>
            <button class="btn">
                Sync
            </button>
        </section>
        <section>
            <SyncPreviewTree :dirPreview="dirPreview" :dirName="dstDir"></SyncPreviewTree>
        </section>
    </main>
</template>

<style scoped>
main {
    width: 100%;
    display: flex;
    flex-direction: column;
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
</style>
