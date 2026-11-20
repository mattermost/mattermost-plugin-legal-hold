import GroupTypes from 'mattermost-redux/action_types/groups';

import {getMissingGroupsByIds} from './groups';

import Client from '@/client';

jest.mock('@/client', () => ({
    __esModule: true,
    default: {getGroup: jest.fn()},
}));

beforeEach(() => {
    jest.spyOn(console, 'log').mockImplementation(() => {});
});

const clientError = (status: number) => Object.assign(new Error('request failed'), {status_code: status});

const getGroup = Client.getGroup as jest.Mock;

const run = (ids: string[], stored: Record<string, unknown> = {}) => {
    const dispatch = jest.fn();
    const getState = () => ({entities: {groups: {groups: stored}}});
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const result = (getMissingGroupsByIds(ids) as any)(dispatch, getState);
    return {dispatch, result};
};

describe('getMissingGroupsByIds', () => {
    test('skips the groups that the store holds', async () => {
        const {dispatch, result} = run(['g1'], {g1: {id: 'g1'}});

        expect(await result).toEqual({data: []});
        expect(getGroup).not.toHaveBeenCalled();
        expect(dispatch).not.toHaveBeenCalled();
    });

    test('stores the groups that load when another group fails', async () => {
        getGroup.mockImplementation((id: string) => (id === 'g2' ? Promise.reject(new Error('404')) : Promise.resolve({id})));
        const {dispatch, result} = run(['g1', 'g2', 'g3']);

        expect(await result).toEqual({error: expect.any(Error)});
        expect(dispatch).toHaveBeenCalledWith({type: GroupTypes.RECEIVED_GROUPS, data: [{id: 'g1'}, {id: 'g3'}]});
    });

    test('shares one request, and its failure, between two callers', async () => {
        let fail: (error: Error) => void = () => {};
        getGroup.mockReturnValue(new Promise((_, reject) => {
            fail = reject;
        }));
        const first = run(['g1']);
        const second = run(['g1']);

        fail(new Error('404'));

        expect(await first.result).toEqual({error: expect.any(Error)});
        expect(await second.result).toEqual({error: expect.any(Error)});
        expect(getGroup).toHaveBeenCalledTimes(1);
    });

    test('requests a failed group again on the next call', async () => {
        getGroup.mockRejectedValueOnce(new Error('404')).mockResolvedValueOnce({id: 'g1'});

        expect(await run(['g1']).result).toEqual({error: expect.any(Error)});
        expect(await run(['g1']).result).toEqual({data: [{id: 'g1'}]});
    });

    test('skips a group that does not exist without an error', async () => {
        getGroup.mockImplementation((id: string) => (id === 'g2' ? Promise.reject(clientError(404)) : Promise.resolve({id})));
        const {dispatch, result} = run(['g1', 'g2']);

        expect(await result).toEqual({data: [{id: 'g1'}]});
        expect(dispatch).toHaveBeenCalledWith({type: GroupTypes.RECEIVED_GROUPS, data: [{id: 'g1'}]});
    });

    test('returns an error for a failure that is not a missing group', async () => {
        const error = clientError(500);
        getGroup.mockRejectedValue(error);

        expect(await run(['g1']).result).toEqual({error});
    });
});
