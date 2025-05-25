export namespace sync {
	
	export enum SyncStatus {
	    None = "None",
	    Created = "Created",
	    Modified = "Modified",
	    Deleted = "Deleted",
	}
	export class SyncPreview {
	    Status: SyncStatus;
	    Subdirs: Record<string, SyncPreview>;
	    Files: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new SyncPreview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Status = source["Status"];
	        this.Subdirs = this.convertValues(source["Subdirs"], SyncPreview, true);
	        this.Files = source["Files"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

