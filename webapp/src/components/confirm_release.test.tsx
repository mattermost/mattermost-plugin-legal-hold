import {render, screen} from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import ConfirmRelease from './confirm_release';

import type {LegalHold} from '@/types';

const legalHold = {id: 'lh1', display_name: 'Quarterly audit'} as LegalHold;

const setup = (releaseLegalHold: (id: string) => Promise<unknown>) => {
    const onClose = jest.fn();
    render(
        <ConfirmRelease
            legalHold={legalHold}
            releaseLegalHold={releaseLegalHold}
            onClose={onClose}
            visible={true}
        />,
    );
    return {onClose};
};

describe('ConfirmRelease', () => {
    test('names the legal hold in the dialog and releases it on confirm', async () => {
        const release = jest.fn().mockResolvedValue({});
        const {onClose} = setup(release);

        const dialog = screen.getByRole('dialog', {name: 'Release legal hold'});
        expect(dialog).toHaveTextContent('"Quarterly audit"');

        await userEvent.click(screen.getByRole('button', {name: 'Release'}));

        expect(release).toHaveBeenCalledWith('lh1');
        expect(onClose).toHaveBeenCalled();
    });

    test('shows a server error and keeps the dialog open', async () => {
        const release = jest.fn().mockRejectedValue(new Error('boom'));
        const {onClose} = setup(release);

        await userEvent.click(screen.getByRole('button', {name: 'Release'}));

        expect(await screen.findByRole('alert')).toHaveTextContent('boom');
        expect(onClose).not.toHaveBeenCalled();
    });

    test('closes without releasing on cancel', async () => {
        const release = jest.fn();
        const {onClose} = setup(release);

        await userEvent.click(screen.getByRole('button', {name: 'Cancel'}));

        expect(release).not.toHaveBeenCalled();
        expect(onClose).toHaveBeenCalled();
    });
});
