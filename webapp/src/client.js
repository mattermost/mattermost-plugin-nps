export class Client {
    constructor() {
        this.url = '/plugins/com.mattermost.nps/api/v1';
    }

    connected = () => {
        fetch(`${this.url}/connected`, {
            method: 'POST',
            headers: {
                'X-Requested-With': 'XMLHttpRequest',
            },
        });
    }
}