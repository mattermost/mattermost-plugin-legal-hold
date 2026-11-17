import {render, screen} from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import ShowSecretModal from './show_secret_modal';

import type {LegalHold} from '@/types';

describe('ShowSecretModal', () => {
    test('shows the secret and the processor flag', () => {
        render(
            <ShowSecretModal
                legalHold={{secret: 'abc123'} as LegalHold}
                onClose={jest.fn()}
                visible={true}
            />,
        );

        const dialog = screen.getByRole('dialog', {name: 'Legal hold secret'});
        expect(dialog).toHaveTextContent('abc123');
        expect(dialog).toHaveTextContent('--legal-hold-secret abc123');
    });

    test('closes', async () => {
        const onClose = jest.fn();
        render(
            <ShowSecretModal
                legalHold={{secret: 'abc123'} as LegalHold}
                onClose={onClose}
                visible={true}
            />,
        );

        // The header has a close button too, so take the one in the footer.
        await userEvent.click(screen.getAllByRole('button', {name: 'Close'}).at(-1)!);

        expect(onClose).toHaveBeenCalled();
    });

    test('renders nothing while hidden', () => {
        render(
            <ShowSecretModal
                legalHold={null}
                onClose={jest.fn()}
                visible={false}
            />,
        );

        expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    });
});
