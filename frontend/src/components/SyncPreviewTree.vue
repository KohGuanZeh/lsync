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

<!-- Need to add key for v-for -->
<template>
    <ul>
        <li>
            <details open>
                <summary :class="getClassFromSyncStatus(dirPreview?.Status)">{{ dirName }}</summary>
                <div class="list-children">
                    <SyncPreviewTree v-for="(v, k) in dirPreview?.Subdirs" :dir-name="k" :dir-preview="v">
                    </SyncPreviewTree>
                    <ul v-if="dirPreview && Object.keys(dirPreview.Files).length > 0">
                        <li v-for="(v, k) in dirPreview?.Files" :class="getClassFromSyncStatus(v)">{{ k }}</li>
                    </ul>
                </div>
            </details>
        </li>
    </ul>
</template>

<style scoped>
ul {
    flex: 1;
    list-style: none;
    padding-left: 0;
    margin: 0.5rem 0;
}

.list-children {
    padding-left: 1rem;
}

.modified {
    color: #ffbf00;
}

.created {
    color: #00a515;
}

.deleted {
    color: #d20000;
}
</style>