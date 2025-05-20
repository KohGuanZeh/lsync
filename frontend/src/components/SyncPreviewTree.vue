<script lang="ts" setup>
import { dirsyncmap } from '../../wailsjs/go/models';

defineProps({
    dirName: String,
    dirPreview: dirsyncmap.DirSyncStruct
})

function getClassFromSyncStatus(status?: string): string {
    if (!status) {
        return ""
    }
    switch (status) {
        case dirsyncmap.SyncStatus.Created:
            return "created";
        case dirsyncmap.SyncStatus.Modified:
            return "modified";
        case dirsyncmap.SyncStatus.Deleted:
            return "deleted";
    }
    return ""
}
</script>

<template>
    <ul>
        <li>
            <details open>
                <summary :class="getClassFromSyncStatus(dirPreview?.Status)">{{ dirName }}</summary>
                <SyncPreviewTree v-for="(v, k) in dirPreview?.Subdirs" :dir-name="k" :dir-preview="v">
                </SyncPreviewTree>
                <ul>
                    <li v-for="(v, k) in dirPreview?.Files" :class="getClassFromSyncStatus(v)">{{ k }}</li>
                </ul>
            </details>
        </li>
    </ul>
</template>

<style scoped>
.modified {
    background-color: #ffbf0077;
    color: #ffbf00;
}

.created {
    background-color: #00800077;
    color: #008000;
}

.deleted {
    background-color: #ba000077;
    color: #ba0000;
}
</style>