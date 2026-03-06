
// this file is generated — do not edit it


declare module "svelte/elements" {
	export interface HTMLAttributes<T> {
		'data-sveltekit-keepfocus'?: true | '' | 'off' | undefined | null;
		'data-sveltekit-noscroll'?: true | '' | 'off' | undefined | null;
		'data-sveltekit-preload-code'?:
			| true
			| ''
			| 'eager'
			| 'viewport'
			| 'hover'
			| 'tap'
			| 'off'
			| undefined
			| null;
		'data-sveltekit-preload-data'?: true | '' | 'hover' | 'tap' | 'off' | undefined | null;
		'data-sveltekit-reload'?: true | '' | 'off' | undefined | null;
		'data-sveltekit-replacestate'?: true | '' | 'off' | undefined | null;
	}
}

export {};


declare module "$app/types" {
	export interface AppTypes {
		RouteId(): "/" | "/auth" | "/auth/login" | "/dashboard" | "/dashboard/agents" | "/dashboard/alerts" | "/dashboard/events" | "/dashboard/rules" | "/sidebar-08";
		RouteParams(): {
			
		};
		LayoutParams(): {
			"/": Record<string, never>;
			"/auth": Record<string, never>;
			"/auth/login": Record<string, never>;
			"/dashboard": Record<string, never>;
			"/dashboard/agents": Record<string, never>;
			"/dashboard/alerts": Record<string, never>;
			"/dashboard/events": Record<string, never>;
			"/dashboard/rules": Record<string, never>;
			"/sidebar-08": Record<string, never>
		};
		Pathname(): "/" | "/auth" | "/auth/" | "/auth/login" | "/auth/login/" | "/dashboard" | "/dashboard/" | "/dashboard/agents" | "/dashboard/agents/" | "/dashboard/alerts" | "/dashboard/alerts/" | "/dashboard/events" | "/dashboard/events/" | "/dashboard/rules" | "/dashboard/rules/" | "/sidebar-08" | "/sidebar-08/";
		ResolvedPathname(): `${"" | `/${string}`}${ReturnType<AppTypes['Pathname']>}`;
		Asset(): "/robots.txt" | string & {};
	}
}