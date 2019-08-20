import {connected} from './actions';

describe('connected', () => {
    const client = {
        connected: jest.fn(),
    };

    test('should call connected API if the user is logged in', () => {
        const getState = () => ({
            entities: {
                users: {
                    currentUserId: 'user1',
                },
            },
        });

        connected(client)(null, getState);

        expect(client.connected).toHaveBeenCalled();
    });

    test('should do nothing if the user is not logged in', () => {
        const getState = () => ({
            entities: {
                users: {
                    currentUserId: '',
                },
            },
        });

        connected(client)(null, getState);

        expect(client.connected).not.toHaveBeenCalled();
    });
});
