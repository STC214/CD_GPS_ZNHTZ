/* Firefox Stealth Final Version v2.1
   Optimized for Cloudflare (CF) and mainstream fingerprint-detection sites.
*/
(() => {
    // 1. Deeply intercept WebDriver using a prototype-chain lock strategy.
    const maskWebDriver = () => {
        try {
            const proto = Navigator.prototype;
            // Remove the native definition and lock the replacement.
            delete proto.webdriver;
            Object.defineProperty(proto, 'webdriver', {
                get: () => false,
                enumerable: true,
                configurable: false
            });
            // Clear any property attached directly to navigator.
            if (Object.getOwnPropertyDescriptor(navigator, 'webdriver')) {
                delete navigator.webdriver;
            }
        } catch (e) {}
    };

    // 2. Mask the plugin array to resemble a real Firefox PluginArray.
    const maskPlugins = () => {
        try {
            const mockPlugins = [
                { name: 'PDF Viewer', filename: 'internal-pdf-viewer', description: 'Portable Document Format' },
                { name: 'Chrome PDF Viewer', filename: 'internal-pdf-viewer', description: 'Google Chrome PDF Viewer' }
            ];

            const fakePlugins = Object.create(PluginArray.prototype);
            
            mockPlugins.forEach((p, i) => {
                const plugin = Object.create(Plugin.prototype);
                Object.defineProperties(plugin, {
                    name: { get: () => p.name, enumerable: true },
                    filename: { get: () => p.filename, enumerable: true },
                    description: { get: () => p.description, enumerable: true }
                });
                fakePlugins[i] = plugin;
                fakePlugins[p.name] = plugin;
            });

            Object.defineProperties(fakePlugins, {
                length: { get: () => mockPlugins.length, enumerable: true },
                item: { value: (i) => fakePlugins[i], writable: false, enumerable: true },
                namedItem: { value: (n) => fakePlugins[n], writable: false, enumerable: true },
                refresh: { value: () => {}, writable: false, enumerable: true }
            });

            // Override the navigator.plugins object as a whole.
            Object.defineProperty(navigator, 'plugins', {
                get: () => fakePlugins,
                enumerable: true,
                configurable: true
            });
        } catch (e) {}
    };

    // 3. Mask basic environment signals such as language, platform, and hardware concurrency.
    const maskEnvironment = () => {
        try {
            Object.defineProperty(navigator, 'languages', { get: () => ['zh-CN', 'zh', 'en-US', 'en'], configurable: true });
            Object.defineProperty(navigator, 'platform', { get: () => 'Win32', configurable: true });
            Object.defineProperty(navigator, 'hardwareConcurrency', { get: () => 8, configurable: true });
            Object.defineProperty(navigator, 'buildID', { get: () => '20181001010101', configurable: true });
        } catch (e) {}
    };

    // 4. Remove Playwright and automation residue markers.
    const cleanAutomationTraces = () => {
        try {
            delete window.__playwright;
            delete window.__pw_cleanup;
            delete window.__PW_inspect;
            // Simulate the absence of the Chrome object, which is the native Firefox state.
            delete window.chrome; 
        } catch (e) {}
    };

    // Execute in order.
    maskWebDriver();
    maskPlugins();
    maskEnvironment();
    cleanAutomationTraces();

    // Keep the console silent so the stealth patch does not emit detectable logs.
})();
