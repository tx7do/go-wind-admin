import {AppRouter} from '@/router';
import {useLocaleSync} from '@go-wind/admin-react';

function App() {
    // 同步 preferences 和 i18n 语言
    useLocaleSync();

    return <AppRouter />;
}

export default App;
