import {Client4, ClientError} from '@mattermost/client';
import {manifest} from './manifest';
import {CreateLegalHold} from "@/types";

class APIClient {
    private readonly url = `/plugins/${manifest.id}/api/v1`;
    private readonly client4 = new Client4();

    getLegalHolds = () => {
        const url = `${this.url}/legalhold/list`;
        return this.doGet(url);
    };

    createLegalHold = (data: CreateLegalHold) => {
        const url = `${this.url}/legalhold/create`;
        return this.doPost(url, data);
    }

    releaseLegalHold = (id: string) => {
        const url = `${this.url}/legalhold/${id}/release`;
        return this.doPost(url, {});
    }

    doGet = async (url: string, headers = {}) => {
        const options = {
            method: 'get',
            headers,
        };

        const response = await fetch(url, this.client4.getOptions(options));

        if (response.ok) {
            return response.json();
        }

        const text = await response.text();

        throw new ClientError(this.client4.url, {
            message: text || '',
            status_code: response.status,
            url,
        });
    };

    doPost = async (url: string, body: any, headers = {}) => {
        const options = {
            method: 'post',
            body: JSON.stringify(body),
            headers,
        };

        const response = await fetch(url, this.client4.getOptions(options));

        if (response.ok) {
            return response.json();
        }

        const text = await response.text();

        throw new ClientError(this.client4.url, {
            message: text || '',
            status_code: response.status,
            url,
        });
    };

}

const Client = new APIClient();
export default Client;
