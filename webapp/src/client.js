import {Client4} from 'mattermost-redux/client';
import {manifest} from './manifest'
import {ClientError} from "@mattermost/client";

class Client {
    constructor() {
        this.url = `/plugins/${manifest.id}/api/v1`;
    };

    listLegalHolds = () => {
        const url = `${this.url}/legalhold/list`;
        return this.doGet(url);
    };

    doGet = async (url, headers = {}) => {
        const options = {
            method: 'get',
            headers,
        };

        const response = await fetch(url, Client4.getOptions(options));

        if (response.ok) {
            return response.json();
        }

        const text = await response.text();

        throw new ClientError(Client4.url, {
            message: text || '',
            status_code: response.status,
            url,
        });
    };
}

export default Client = new Client();
