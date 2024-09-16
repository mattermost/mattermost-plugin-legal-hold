import {Client4, ClientError} from '@mattermost/client';

import {CreateLegalHold, UpdateLegalHold} from '@/types';

import {manifest} from './manifest';

class APIClient {
    private readonly url = `/plugins/${manifest.id}/api/v1`;
    private readonly client4 = new Client4();

    downloadUrl = (id: string) => {
        return `${this.url}/legalhold/${id}/download`;
    };

    getLegalHolds = () => {
        const url = `${this.url}/legalhold`;
        return this.doGet(url);
    };

    createLegalHold = (data: CreateLegalHold) => {
        const url = `${this.url}/legalhold`;
        return this.doWithBody(url, 'post', data);
    };

    releaseLegalHold = (id: string) => {
        const url = `${this.url}/legalhold/${id}/release`;
        return this.doWithBody(url, 'post', {});
    };

    updateLegalHold = (id: string, data: UpdateLegalHold) => {
        const url = `${this.url}/legalhold/${id}`;
        return this.doWithBody(url, 'put', data);
    };

    testAmazonS3Connection = () => {
        const url = `${this.url}/test_amazon_s3_connection`;
        return this.doWithBody(url, 'post', {}) as Promise<{message: string}>;
    };

    private doGet = async (url: string, headers = {}) => {
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

    private doWithBody = async (url: string, method: string, body: any, headers = {}) => {
        const options = {
            method,
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
